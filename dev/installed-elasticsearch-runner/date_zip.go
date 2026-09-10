package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
)

const maximumDateArchive = 256 << 20

// archiveProjection supports the observed Gradle ZIP32 form. It preserves raw
// framing and compressed non-manifest payloads, allowing only validated date
// changes and their derived CRC, size and subsequent entry offsets. Unsupported
// framing fails closed; this never rewrites a captured archive.
func archiveProjection(raw []byte, allowed []string, window nativeTimeWindow) (string, error) {
	if len(raw) == 0 || len(raw) > maximumDateArchive || len(allowed) == 0 {
		return "", errors.New("archive or manifest allowlist outside bounds")
	}
	archive, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return "", err
	}
	if len(archive.File) == 0 || len(archive.File) >= 65535 {
		return "", errors.New("ZIP32 entry count required")
	}
	approved := map[string]bool{}
	for _, name := range allowed {
		if approved[name] {
			return "", errors.New("duplicate approved manifest")
		}
		approved[name] = true
	}
	names := map[string]bool{}
	var expanded uint64
	for _, f := range archive.File {
		if names[f.Name] || f.Name == "" || strings.HasPrefix(f.Name, "/") || strings.ContainsAny(f.Name, "\\\x00") || path.Clean(f.Name) != strings.TrimSuffix(f.Name, "/") || f.Name == ".." || strings.HasPrefix(f.Name, "../") {
			return "", errors.New("duplicate or unsafe archive entry")
		}
		names[f.Name] = true
		if f.UncompressedSize64 > maximumDateArchive || expanded+f.UncompressedSize64 > maximumDateArchive {
			return "", errors.New("expanded archive exceeds limit")
		}
		expanded += f.UncompressedSize64
	}
	for name := range approved {
		if !names[name] {
			return "", errors.New("approved manifest absent")
		}
	}
	// ZIP32's end record is fixed-size except its already decoded comment.
	endOffset := len(raw) - 22 - len(archive.Comment)
	end, err := zipBytes(raw, endOffset, 22)
	if err != nil || string(end[:4]) != "PK\x05\x06" {
		return "", errors.New("ZIP32 end record required")
	}
	le := binary.LittleEndian
	centralStart := int(le.Uint32(end[16:20]))
	centralSize := int(le.Uint32(end[12:16]))
	if le.Uint16(end[4:6]) != 0 || le.Uint16(end[6:8]) != 0 || int(le.Uint16(end[8:10])) != len(archive.File) || int(le.Uint16(end[10:12])) != len(archive.File) || int(le.Uint16(end[20:22])) != len(archive.Comment) || centralStart+centralSize != endOffset || string(raw[endOffset+22:]) != archive.Comment {
		return "", errors.New("inconsistent or spanned ZIP end record")
	}
	h := sha256.New()
	add := func(value []byte) {
		var size [8]byte
		le.PutUint64(size[:], uint64(len(value)))
		h.Write(size[:])
		h.Write(value)
	}
	local, central := 0, centralStart
	for _, f := range archive.File {
		lh, e := zipBytes(raw, local, 30)
		if e != nil || string(lh[:4]) != "PK\x03\x04" {
			return "", errors.New("invalid local header or archive prefix")
		}
		ch, e := zipBytes(raw, central, 46)
		if e != nil || string(ch[:4]) != "PK\x01\x02" {
			return "", errors.New("invalid central header")
		}
		if !bytes.Equal(lh[4:14], ch[6:16]) || f.ReaderVersion >= 45 || f.Flags & ^uint16(0x808) != 0 || (f.Method != zip.Store && f.Method != zip.Deflate) {
			return "", errors.New("unsupported ZIP metadata")
		}
		if le.Uint32(ch[16:20]) != f.CRC32 || uint64(le.Uint32(ch[20:24])) != f.CompressedSize64 || uint64(le.Uint32(ch[24:28])) != f.UncompressedSize64 || int(le.Uint32(ch[42:46])) != local {
			return "", errors.New("central CRC, size or offset mismatch")
		}
		ln, lx := int(le.Uint16(lh[26:28])), int(le.Uint16(lh[28:30]))
		cn, cx, cc := int(le.Uint16(ch[28:30])), int(le.Uint16(ch[30:32])), int(le.Uint16(ch[32:34]))
		localName, e := zipBytes(raw, local+30, ln)
		if e != nil {
			return "", e
		}
		centralName, e := zipBytes(raw, central+46, cn)
		if e != nil {
			return "", e
		}
		extra, e := zipBytes(raw, local+30+ln, lx)
		if e != nil {
			return "", e
		}
		suffix, e := zipBytes(raw, central+46, cn+cx+cc)
		if e != nil {
			return "", e
		}
		if !bytes.Equal(localName, centralName) || string(localName) != f.Name || !bytes.Equal(extra, f.Extra) {
			return "", errors.New("ZIP name or local metadata differs")
		}
		begin := local + 30 + ln + lx
		finish := begin + int(f.CompressedSize64)
		offset, e := f.DataOffset()
		if e != nil || offset != int64(begin) || finish > centralStart {
			return "", errors.New("ZIP payload offset mismatch")
		}
		compressed, e := zipBytes(raw, begin, int(f.CompressedSize64))
		if e != nil {
			return "", e
		}
		crcSizes := make([]byte, 12)
		le.PutUint32(crcSizes, f.CRC32)
		le.PutUint32(crcSizes[4:], uint32(f.CompressedSize64))
		le.PutUint32(crcSizes[8:], uint32(f.UncompressedSize64))
		descriptor := []byte{}
		if f.Flags&8 != 0 {
			descriptor, e = zipBytes(raw, finish, 16)
			if e != nil || !bytes.Equal(lh[14:26], make([]byte, 12)) || string(descriptor[:4]) != "PK\x07\x08" || !bytes.Equal(descriptor[4:], crcSizes) {
				return "", errors.New("invalid ZIP data descriptor")
			}
		} else if !bytes.Equal(lh[14:26], crcSizes) {
			return "", errors.New("local CRC or size mismatch")
		}
		reader, e := f.Open()
		if e != nil {
			return "", e
		}
		payload, e := io.ReadAll(io.LimitReader(reader, int64(f.UncompressedSize64)+1))
		closeErr := reader.Close()
		if e != nil || closeErr != nil || uint64(len(payload)) != f.UncompressedSize64 {
			return "", fmt.Errorf("invalid ZIP payload: %s", f.Name)
		}
		if approved[f.Name] {
			payload, e = manifestProjection(payload, window)
			if e != nil {
				return "", fmt.Errorf("%s: %w", f.Name, e)
			}
			clear(lh[14:26])
			clear(ch[16:28])
			if len(descriptor) != 0 {
				clear(descriptor[4:])
			}
		} else {
			payload = compressed
		}
		clear(ch[42:46])
		for _, value := range [][]byte{lh, localName, extra, payload, descriptor, ch, suffix} {
			add(value)
		}
		local = finish + len(descriptor)
		central += 46 + cn + cx + cc
	}
	if local != centralStart || central != endOffset {
		return "", errors.New("ZIP padding, prefix or trailing records")
	}
	clear(end[16:20])
	add(end)
	add([]byte(archive.Comment))
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Return an owned segment because derived header fields are masked for hashing.
func zipBytes(raw []byte, offset, size int) ([]byte, error) {
	if offset < 0 || size < 0 || offset > len(raw) || size > len(raw)-offset {
		return nil, errors.New("truncated ZIP structure")
	}
	return bytes.Clone(raw[offset : offset+size]), nil
}
