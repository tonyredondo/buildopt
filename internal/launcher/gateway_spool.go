package launcher

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	gatewayMaximumVerifiedPayloadBytes int64 = 100 << 20
	gatewayVerifiedSpoolQuotaBytes     int64 = 200 << 20
	gatewaySpoolFilePrefix                   = ".get-"
	gatewayDirectorySyncBatchWindow          = 2 * time.Millisecond
)

var errGatewaySpoolUnavailable = errors.New(
	"local gateway verified spool is unavailable",
)

var (
	errGatewaySpoolFlood = errors.New(
		"local gateway verified spool capacity is exhausted",
	)
	errGatewaySpoolObjectTooLarge = errors.New(
		"local gateway verified object exceeds the maximum size",
	)
	errGatewaySpoolDiskPressure = errors.New(
		"local gateway verified spool has no disk capacity",
	)
)

type gatewaySpool struct {
	root                       string
	maximumBytes               int64
	quotaBytes                 int64
	removeRoot                 bool
	mutex                      sync.Mutex
	reservedBytes              int64
	closed                     bool
	directoryMutex             sync.Mutex
	directoryBatch             *gatewayDirectorySyncBatch
	testWrite                  func(*os.File, []byte) (int, error)
	testAfterReserve           func()
	testAfterDirectorySyncJoin func(bool)
	testSyncDirectory          func(string) error
}

type gatewayDirectorySyncBatch struct {
	done chan struct{}
	err  error
}

type gatewaySpoolReservation struct {
	spool    *gatewaySpool
	reserved int64
	released bool
}

type verifiedGatewayPayload struct {
	file        *os.File
	reservation *gatewaySpoolReservation
	size        int64
	digest      string
}

func newEphemeralGatewaySpool() (*gatewaySpool, error) {
	root, err := os.MkdirTemp("", "buildopt-gateway-spool-")
	if err != nil {
		return nil, fmt.Errorf("create local gateway spool: %w", err)
	}
	spool, err := openGatewaySpool(
		root,
		true,
		gatewayMaximumVerifiedPayloadBytes,
		gatewayVerifiedSpoolQuotaBytes,
	)
	if err != nil {
		_ = os.Remove(root)
		return nil, err
	}
	return spool, nil
}

func openGatewaySpool(
	root string,
	removeRoot bool,
	maximumBytes int64,
	quotaBytes int64,
) (*gatewaySpool, error) {
	if !filepath.IsAbs(root) ||
		filepath.Clean(root) != root ||
		maximumBytes < 1 ||
		quotaBytes < maximumBytes {
		return nil, errors.New("invalid local gateway spool configuration")
	}
	if err := ensurePrivateDirectory(root, true); err != nil {
		return nil, fmt.Errorf("prepare local gateway spool: %w", err)
	}
	spool := &gatewaySpool{
		root:         root,
		maximumBytes: maximumBytes,
		quotaBytes:   quotaBytes,
		removeRoot:   removeRoot,
	}
	if err := spool.cleanupStaleFiles(); err != nil {
		if removeRoot {
			_ = os.Remove(root)
		}
		return nil, err
	}
	return spool, nil
}

func (spool *gatewaySpool) receive(
	ctx context.Context,
	body io.Reader,
	contentLength int64,
	etag string,
	digestHeader string,
) (verifiedGatewayPayload, error) {
	if ctx == nil || body == nil {
		return verifiedGatewayPayload{}, errGatewaySpoolUnavailable
	}
	expectedDigest, err := gatewayDigestFromHeaders(etag, digestHeader)
	if err != nil {
		return verifiedGatewayPayload{}, err
	}
	reservation, err := spool.reserve(contentLength)
	if err != nil {
		return verifiedGatewayPayload{}, err
	}
	releaseReservation := true
	defer func() {
		if releaseReservation {
			reservation.release()
		}
	}()

	file, err := os.CreateTemp(spool.root, gatewaySpoolFilePrefix+"*")
	if err != nil {
		return verifiedGatewayPayload{}, gatewaySpoolIOError("create", err)
	}
	path := file.Name()
	remove := true
	closeFile := true
	defer func() {
		if closeFile {
			_ = file.Close()
		}
		if remove {
			_ = os.Remove(path)
		}
	}()
	if err := validateGatewaySpoolFile(file); err != nil {
		return verifiedGatewayPayload{}, err
	}

	digest := sha256.New()
	writer := &gatewaySpoolWriter{
		spool:       spool,
		reservation: reservation,
		file:        file,
	}
	size, err := io.Copy(
		io.MultiWriter(writer, digest),
		io.LimitReader(
			&gatewayContextReader{ctx: ctx, reader: body},
			spool.maximumBytes+1,
		),
	)
	if err != nil {
		if errors.Is(err, errGatewaySpoolObjectTooLarge) ||
			errors.Is(err, errGatewaySpoolFlood) ||
			errors.Is(err, errGatewaySpoolDiskPressure) {
			return verifiedGatewayPayload{}, err
		}
		return verifiedGatewayPayload{}, fmt.Errorf(
			"%w: write: %w",
			errGatewaySpoolUnavailable,
			err,
		)
	}
	if size > spool.maximumBytes {
		return verifiedGatewayPayload{}, errors.Join(
			errGatewaySpoolUnavailable,
			errGatewaySpoolObjectTooLarge,
		)
	}
	if contentLength >= 0 && size != contentLength {
		return verifiedGatewayPayload{}, errGatewaySpoolUnavailable
	}
	if err := ctx.Err(); err != nil {
		return verifiedGatewayPayload{}, fmt.Errorf(
			"%w: %v",
			errGatewaySpoolUnavailable,
			err,
		)
	}
	actualDigest := "sha256:" + hex.EncodeToString(digest.Sum(nil))
	if actualDigest != expectedDigest {
		return verifiedGatewayPayload{}, errGatewaySpoolUnavailable
	}
	if err := file.Sync(); err != nil {
		return verifiedGatewayPayload{}, gatewaySpoolIOError("sync", err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return verifiedGatewayPayload{}, fmt.Errorf(
			"%w: rewind: %v",
			errGatewaySpoolUnavailable,
			err,
		)
	}
	if err := os.Remove(path); err != nil {
		return verifiedGatewayPayload{}, gatewaySpoolIOError("unlink", err)
	}
	remove = false
	if err := spool.syncUnlinkedDirectory(); err != nil {
		return verifiedGatewayPayload{}, gatewaySpoolIOError(
			"sync directory",
			err,
		)
	}
	closeFile = false
	releaseReservation = false
	return verifiedGatewayPayload{
		file:        file,
		reservation: reservation,
		size:        size,
		digest:      actualDigest,
	}, nil
}

func (spool *gatewaySpool) syncUnlinkedDirectory() error {
	spool.directoryMutex.Lock()
	batch := spool.directoryBatch
	leader := batch == nil
	if leader {
		batch = &gatewayDirectorySyncBatch{done: make(chan struct{})}
		spool.directoryBatch = batch
	}
	spool.directoryMutex.Unlock()
	if spool.testAfterDirectorySyncJoin != nil {
		spool.testAfterDirectorySyncJoin(leader)
	}

	if leader {
		timer := time.NewTimer(gatewayDirectorySyncBatchWindow)
		<-timer.C
		spool.directoryMutex.Lock()
		if spool.directoryBatch == batch {
			spool.directoryBatch = nil
		}
		spool.directoryMutex.Unlock()
		if spool.testSyncDirectory != nil {
			batch.err = spool.testSyncDirectory(spool.root)
		} else {
			batch.err = syncGatewaySpoolDirectory(spool.root)
		}
		close(batch.done)
	} else {
		<-batch.done
	}
	return batch.err
}

func (payload *verifiedGatewayPayload) close() error {
	if payload == nil {
		return nil
	}
	var err error
	if payload.file != nil {
		err = payload.file.Close()
		payload.file = nil
	}
	if payload.reservation != nil {
		payload.reservation.release()
		payload.reservation = nil
	}
	return err
}

func gatewayDigestFromHeaders(
	etag string,
	digestHeader string,
) (string, error) {
	if len(etag) < 3 ||
		etag[0] != '"' ||
		etag[len(etag)-1] != '"' {
		return "", errGatewaySpoolUnavailable
	}
	digest := etag[1 : len(etag)-1]
	if !validGatewayAuthorityDigest(digest) ||
		digestHeader != "" && digestHeader != digest {
		return "", errGatewaySpoolUnavailable
	}
	return digest, nil
}

func (spool *gatewaySpool) reserve(
	contentLength int64,
) (*gatewaySpoolReservation, error) {
	reserved := contentLength
	if reserved < 0 {
		reserved = spool.maximumBytes
	}
	if reserved > spool.maximumBytes {
		return nil, errors.Join(
			errGatewaySpoolUnavailable,
			errGatewaySpoolObjectTooLarge,
		)
	}
	spool.mutex.Lock()
	if spool.closed {
		spool.mutex.Unlock()
		return nil, errGatewaySpoolUnavailable
	}
	if reserved > spool.quotaBytes-spool.reservedBytes {
		spool.mutex.Unlock()
		return nil, errors.Join(
			errGatewaySpoolUnavailable,
			errGatewaySpoolFlood,
		)
	}
	spool.reservedBytes += reserved
	spool.mutex.Unlock()

	reservation := &gatewaySpoolReservation{
		spool:    spool,
		reserved: reserved,
	}
	if spool.testAfterReserve != nil {
		spool.testAfterReserve()
	}
	return reservation, nil
}

func (reservation *gatewaySpoolReservation) ensure(size int64) error {
	if size <= reservation.reserved {
		return nil
	}
	spool := reservation.spool
	spool.mutex.Lock()
	defer spool.mutex.Unlock()
	additional := size - reservation.reserved
	if reservation.released || spool.closed {
		return errGatewaySpoolUnavailable
	}
	if size > spool.maximumBytes {
		return errors.Join(
			errGatewaySpoolUnavailable,
			errGatewaySpoolObjectTooLarge,
		)
	}
	if additional > spool.quotaBytes-spool.reservedBytes {
		return errors.Join(
			errGatewaySpoolUnavailable,
			errGatewaySpoolFlood,
		)
	}
	spool.reservedBytes += additional
	reservation.reserved = size
	return nil
}

func (reservation *gatewaySpoolReservation) release() {
	if reservation == nil || reservation.spool == nil {
		return
	}
	spool := reservation.spool
	spool.mutex.Lock()
	defer spool.mutex.Unlock()
	if reservation.released {
		return
	}
	spool.reservedBytes -= reservation.reserved
	reservation.released = true
}

type gatewaySpoolWriter struct {
	spool       *gatewaySpool
	reservation *gatewaySpoolReservation
	file        *os.File
	written     int64
}

func (writer *gatewaySpoolWriter) Write(content []byte) (int, error) {
	if err := writer.reservation.ensure(
		writer.written + int64(len(content)),
	); err != nil {
		return 0, err
	}
	var (
		written int
		err     error
	)
	if writer.spool.testWrite != nil {
		written, err = writer.spool.testWrite(writer.file, content)
	} else {
		written, err = writer.file.Write(content)
	}
	writer.written += int64(written)
	if err != nil {
		return written, gatewaySpoolIOError("write", err)
	}
	return written, err
}

func gatewaySpoolIOError(operation string, err error) error {
	if errors.Is(err, syscall.ENOSPC) || errors.Is(err, syscall.EDQUOT) {
		return fmt.Errorf(
			"%w: %w: %s: %w",
			errGatewaySpoolUnavailable,
			errGatewaySpoolDiskPressure,
			operation,
			err,
		)
	}
	return fmt.Errorf(
		"%w: %s: %w",
		errGatewaySpoolUnavailable,
		operation,
		err,
	)
}

type gatewayContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (reader *gatewayContextReader) Read(content []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	count, err := reader.reader.Read(content)
	if err == nil {
		if contextErr := reader.ctx.Err(); contextErr != nil {
			return count, contextErr
		}
	}
	return count, err
}

func validateGatewaySpoolFile(file *os.File) error {
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("%w: inspect file: %v", errGatewaySpoolUnavailable, err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok ||
		stat.Uid != uint32(os.Geteuid()) ||
		!info.Mode().IsRegular() ||
		info.Mode().Perm() != 0o600 ||
		stat.Nlink != 1 {
		return errors.New("local gateway spool file is not private")
	}
	return nil
}

func (spool *gatewaySpool) cleanupStaleFiles() error {
	entries, err := os.ReadDir(spool.root)
	if err != nil {
		return fmt.Errorf("inspect local gateway spool: %w", err)
	}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), gatewaySpoolFilePrefix) {
			return fmt.Errorf(
				"local gateway spool has unexpected entry %q",
				entry.Name(),
			)
		}
		path := filepath.Join(spool.root, entry.Name())
		file, err := os.OpenFile(
			path,
			os.O_RDONLY|syscall.O_NOFOLLOW,
			0,
		)
		if err != nil {
			return fmt.Errorf("open stale local gateway spool file: %w", err)
		}
		validateErr := validateGatewaySpoolFile(file)
		closeErr := file.Close()
		if validateErr != nil || closeErr != nil {
			return errors.Join(validateErr, closeErr)
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove stale local gateway spool file: %w", err)
		}
	}
	if len(entries) > 0 {
		if err := syncGatewaySpoolDirectory(spool.root); err != nil {
			return fmt.Errorf("sync cleaned local gateway spool: %w", err)
		}
	}
	return nil
}

func (spool *gatewaySpool) close() error {
	if spool == nil {
		return nil
	}
	spool.mutex.Lock()
	if spool.closed {
		spool.mutex.Unlock()
		return nil
	}
	spool.closed = true
	active := spool.reservedBytes
	spool.mutex.Unlock()
	if active != 0 {
		return errors.New("close local gateway spool with active reservations")
	}
	if err := spool.cleanupStaleFiles(); err != nil {
		return err
	}
	if spool.removeRoot {
		if err := os.Remove(spool.root); err != nil {
			return fmt.Errorf("remove local gateway spool: %w", err)
		}
	}
	return nil
}

func syncGatewaySpoolDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
