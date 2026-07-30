package sharedcache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const digestPrefix = "sha256:"

type filesystemBlobStore struct {
	owner            *Storage
	blobRoot         string
	spoolRoot        string
	maximumBlobBytes int64
	publishLocks     [256]sync.Mutex
}

var _ BlobStore = (*filesystemBlobStore)(nil)

func (store *filesystemBlobStore) Put(
	ctx context.Context,
	reader io.Reader,
) (Blob, bool, error) {
	if ctx == nil {
		return Blob{}, false, errors.New("put Shared blob: nil context")
	}
	if reader == nil {
		return Blob{}, false, errors.New("put Shared blob: nil reader")
	}
	finish, err := store.owner.beginOperation()
	if err != nil {
		return Blob{}, false, err
	}
	defer finish()
	store.owner.reconcileMutex.RLock()
	defer store.owner.reconcileMutex.RUnlock()
	return store.putLocked(ctx, reader)
}

func (store *filesystemBlobStore) putLocked(
	ctx context.Context,
	reader io.Reader,
) (Blob, bool, error) {
	spool, err := os.CreateTemp(store.spoolRoot, ".blob-*")
	if err != nil {
		return Blob{}, false, fmt.Errorf("put Shared blob: create spool: %w", err)
	}
	spoolPath := spool.Name()
	defer func() {
		_ = spool.Close()
		_ = os.Remove(spoolPath)
	}()
	if err := validatePrivateRegularFile(spool); err != nil {
		return Blob{}, false, fmt.Errorf(
			"put Shared blob: validate spool: %w",
			err,
		)
	}

	digest := sha256.New()
	limited := &io.LimitedReader{
		R: &contextReader{ctx: ctx, reader: reader},
		N: store.maximumBlobBytes + 1,
	}
	size, err := io.Copy(io.MultiWriter(spool, digest), limited)
	if err != nil {
		return Blob{}, false, fmt.Errorf("put Shared blob: stream: %w", err)
	}
	if size > store.maximumBlobBytes {
		return Blob{}, false, ErrBlobTooLarge
	}
	if err := ctx.Err(); err != nil {
		return Blob{}, false, fmt.Errorf("put Shared blob: %w", err)
	}
	if err := spool.Sync(); err != nil {
		return Blob{}, false, fmt.Errorf("put Shared blob: sync spool: %w", err)
	}
	if err := spool.Close(); err != nil {
		return Blob{}, false, fmt.Errorf("put Shared blob: close spool: %w", err)
	}

	blob := Blob{
		Digest: digestPrefix + hex.EncodeToString(digest.Sum(nil)),
		Size:   size,
	}
	digestBytes := digest.Sum(nil)
	publishLock := &store.publishLocks[int(digestBytes[0])]
	publishLock.Lock()
	defer publishLock.Unlock()
	if err := ctx.Err(); err != nil {
		return Blob{}, false, fmt.Errorf("put Shared blob: %w", err)
	}

	finalPath, err := store.pathForDigest(blob.Digest, true)
	if err != nil {
		return Blob{}, false, fmt.Errorf("put Shared blob: path: %w", err)
	}
	if err := os.Link(spoolPath, finalPath); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return Blob{}, false, fmt.Errorf(
				"put Shared blob: publish immutable object: %w",
				err,
			)
		}
		file, verifyErr := store.openVerified(ctx, blob)
		if file != nil {
			_ = file.Close()
		}
		if verifyErr != nil {
			return Blob{}, false, verifyErr
		}
		return blob, false, nil
	}
	if err := syncDirectory(filepath.Dir(finalPath)); err != nil {
		return Blob{}, false, fmt.Errorf(
			"put Shared blob: sync digest directory: %w",
			err,
		)
	}
	if err := os.Remove(spoolPath); err != nil {
		return Blob{}, false, fmt.Errorf(
			"put Shared blob: remove published spool link: %w",
			err,
		)
	}
	if err := syncDirectory(store.spoolRoot); err != nil {
		return Blob{}, false, fmt.Errorf(
			"put Shared blob: sync spool directory: %w",
			err,
		)
	}
	return blob, true, nil
}

func (store *filesystemBlobStore) OpenVerified(
	ctx context.Context,
	blob Blob,
) (*os.File, error) {
	if ctx == nil {
		return nil, errors.New("open Shared blob: nil context")
	}
	finish, err := store.owner.beginOperation()
	if err != nil {
		return nil, err
	}
	defer finish()
	store.owner.reconcileMutex.RLock()
	defer store.owner.reconcileMutex.RUnlock()
	return store.openVerified(ctx, blob)
}

func (store *filesystemBlobStore) openVerified(
	ctx context.Context,
	blob Blob,
) (*os.File, error) {
	if blob.Size < 0 || blob.Size > store.maximumBlobBytes {
		return nil, fmt.Errorf(
			"open Shared blob: invalid expected size %d",
			blob.Size,
		)
	}
	path, err := store.pathForDigest(blob.Digest, false)
	if err != nil {
		return nil, fmt.Errorf("open Shared blob: %w", err)
	}
	file, err := openPrivateBlob(path)
	if err != nil {
		return nil, fmt.Errorf("open Shared blob: %w", err)
	}
	digest := sha256.New()
	size, readErr := io.Copy(
		digest,
		&contextReader{ctx: ctx, reader: file},
	)
	if readErr != nil {
		_ = file.Close()
		return nil, fmt.Errorf("open Shared blob: verify: %w", readErr)
	}
	actualDigest := digestPrefix + hex.EncodeToString(digest.Sum(nil))
	if size != blob.Size || actualDigest != blob.Digest {
		_ = file.Close()
		return nil, fmt.Errorf(
			"%w: expected %s/%d, found %s/%d",
			ErrBlobCorrupt,
			blob.Digest,
			blob.Size,
			actualDigest,
			size,
		)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("open Shared blob: rewind: %w", err)
	}
	return file, nil
}

func (store *filesystemBlobStore) pathForDigest(
	digest string,
	createDirectory bool,
) (string, error) {
	hexDigest, err := parseDigest(digest)
	if err != nil {
		return "", err
	}
	directory := filepath.Join(store.blobRoot, hexDigest[:2])
	if createDirectory {
		if err := preparePrivateDirectory(directory); err != nil {
			return "", err
		}
		if err := validateLocalStorageFilesystem(
			store.owner.layout.Root,
			directory,
		); err != nil {
			return "", err
		}
		if err := syncDirectory(store.blobRoot); err != nil {
			return "", err
		}
	}
	return filepath.Join(directory, hexDigest[2:]), nil
}

func parseDigest(digest string) (string, error) {
	if len(digest) != len(digestPrefix)+sha256.Size*2 ||
		!strings.HasPrefix(digest, digestPrefix) {
		return "", ErrInvalidDigest
	}
	hexDigest := strings.TrimPrefix(digest, digestPrefix)
	decoded, err := hex.DecodeString(hexDigest)
	if err != nil ||
		len(decoded) != sha256.Size ||
		hex.EncodeToString(decoded) != hexDigest {
		return "", ErrInvalidDigest
	}
	return hexDigest, nil
}

func (store *filesystemBlobStore) list() ([]Blob, error) {
	shards, err := os.ReadDir(store.blobRoot)
	if err != nil {
		return nil, err
	}
	var blobs []Blob
	for _, shard := range shards {
		if !shard.IsDir() ||
			len(shard.Name()) != 2 ||
			!isLowerHex(shard.Name()) {
			return nil, fmt.Errorf(
				"unexpected blob shard %q",
				shard.Name(),
			)
		}
		shardPath := filepath.Join(store.blobRoot, shard.Name())
		if err := preparePrivateDirectory(shardPath); err != nil {
			return nil, err
		}
		entries, err := os.ReadDir(shardPath)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() ||
				len(entry.Name()) != 62 ||
				!isLowerHex(entry.Name()) {
				return nil, fmt.Errorf(
					"unexpected blob entry %q",
					filepath.Join(shard.Name(), entry.Name()),
				)
			}
			digest := digestPrefix + shard.Name() + entry.Name()
			file, err := openPrivateBlob(
				filepath.Join(shardPath, entry.Name()),
			)
			if err != nil {
				return nil, err
			}
			info, statErr := file.Stat()
			closeErr := file.Close()
			if statErr != nil {
				return nil, statErr
			}
			if closeErr != nil {
				return nil, closeErr
			}
			blobs = append(blobs, Blob{
				Digest: digest,
				Size:   info.Size(),
			})
		}
	}
	return blobs, nil
}

func (store *filesystemBlobStore) remove(blob Blob) error {
	path, err := store.pathForDigest(blob.Digest, false)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

func (store *filesystemBlobStore) quarantine(
	blob Blob,
	reason string,
	now time.Time,
) (bool, error) {
	source, err := store.pathForDigest(blob.Digest, false)
	if err != nil {
		return false, err
	}
	if _, err := os.Lstat(source); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	hexDigest, err := parseDigest(blob.Digest)
	if err != nil {
		return false, err
	}
	base := hexDigest + "." +
		strconv.FormatInt(now.UnixMilli(), 10) + "." +
		strings.ToLower(reason)
	destination := filepath.Join(store.owner.layout.Quarantine, base)
	for suffix := 0; ; suffix++ {
		candidate := destination
		if suffix > 0 {
			candidate += "." + strconv.Itoa(suffix)
		}
		if _, err := os.Lstat(candidate); errors.Is(err, os.ErrNotExist) {
			destination = candidate
			break
		} else if err != nil {
			return false, err
		}
	}
	if err := os.Rename(source, destination); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if err := syncDirectory(filepath.Dir(source)); err != nil {
		return false, err
	}
	if err := syncDirectory(store.owner.layout.Quarantine); err != nil {
		return false, err
	}
	return true, nil
}

func isLowerHex(value string) bool {
	for _, character := range value {
		if (character < '0' || character > '9') &&
			(character < 'a' || character > 'f') {
			return false
		}
	}
	return value != ""
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (reader *contextReader) Read(buffer []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	count, err := reader.reader.Read(buffer)
	if err == nil {
		if contextErr := reader.ctx.Err(); contextErr != nil {
			return count, contextErr
		}
	}
	return count, err
}
