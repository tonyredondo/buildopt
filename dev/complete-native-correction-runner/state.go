package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
)

const stateSchema = "buildopt.cnc/capture-state/v1"

type clockReading struct {
	Boot    string
	Seconds float64
}
type campaign struct {
	Schema        string  `json:"schemaVersion"`
	PackageSHA256 string  `json:"packageSha256"`
	Boot          string  `json:"bootId"`
	Started       float64 `json:"startedBootSeconds"`
	Deadline      float64 `json:"deadlineBootSeconds"`
	ReviewAt      float64 `json:"reviewBootSeconds"`
	MaximumStarts int     `json:"maximumStarts"`
}
type reservation struct {
	Slot          string  `json:"slot"`
	PackageSHA256 string  `json:"packageSha256"`
	BootSeconds   float64 `json:"reservedBootSeconds"`
	RequestSHA256 string  `json:"requestSha256"`
}

func bootClock() (clockReading, error) {
	boot, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return clockReading{}, err
	}
	uptime, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return clockReading{}, err
	}
	fields := strings.Fields(string(uptime))
	if len(fields) != 2 {
		return clockReading{}, errors.New("invalid boot clock")
	}
	seconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return clockReading{}, err
	}
	return clockReading{strings.TrimSpace(string(boot)), seconds}, nil
}

func initialize(root, packageDigest string, now clockReading) error {
	if !filepath.IsAbs(root) || filepath.Clean(root) == "/" {
		return errors.New("state must be a new absolute directory")
	}
	if err := os.Mkdir(root, 0700); err != nil {
		return err
	}
	// Never overwrite or recover an occupied directory automatically.
	state := campaign{stateSchema, packageDigest, now.Boot, now.Seconds, now.Seconds + 7200, now.Seconds + 1800, 60}
	if err := writeNewJSON(filepath.Join(root, "state.json"), state); err != nil {
		return err
	}
	if err := os.Mkdir(filepath.Join(root, "attempts"), 0700); err != nil {
		return err
	}
	file, err := os.OpenFile(filepath.Join(root, "lock"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	return file.Close()
}

func lockCampaign(root string) (*os.File, error) {
	file, err := os.OpenFile(filepath.Join(root, "lock"), os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		return nil, errors.New("campaign already active")
	}
	return file, nil
}

func remaining(state campaign, now clockReading) (float64, error) {
	if err := validateState(state); err != nil {
		return 0, err
	}
	if now.Boot != state.Boot || now.Seconds < state.Started {
		return 0, errors.New("boot identity changed or clock moved backwards; no reset")
	}
	if now.Seconds >= state.Deadline {
		return 0, errors.New("INCOMPLETE_EXPERIMENT_BUDGET_EXHAUSTED")
	}
	return state.Deadline - now.Seconds, nil
}

func validateState(state campaign) error {
	if state.Schema != stateSchema || state.MaximumStarts != 60 || state.Deadline-state.Started != 7200 || state.ReviewAt-state.Started != 1800 || state.Boot == "" || state.Started < 0 || len(state.PackageSHA256) != 64 {
		return errors.New("state limit or identity drift")
	}
	return nil
}

func readJSON(path string, target any) error {
	parent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return err
	}
	if parent != filepath.Clean(filepath.Dir(path)) {
		return errors.New("symlink ancestor in JSON evidence")
	}
	if err := contained(filepath.Dir(path), path); err != nil {
		return err
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, (4<<20)+1))
	if err != nil {
		return err
	}
	if len(data) > 4<<20 {
		return errors.New("JSON exceeds 4 MiB")
	}
	return decodeDocument(data, target)
}

func decodeDocument(data []byte, target any) error {
	if _, err := contractcrypto.CanonicalizeJCS(data); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := d.Decode(&trailing); err != io.EOF {
		return errors.New("trailing JSON")
	}
	return nil
}

func writeNewJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeNew(path, append(data, '\n'))
}

// Link publishes a complete file without replacing an occupied destination.
func writeNew(path string, data []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".pending-")
	if err != nil {
		return err
	}
	temp := file.Name()
	defer os.Remove(temp)
	if _, err = file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	if err = os.Link(temp, path); err != nil {
		return err
	}
	directory, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func fileDigest(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("expected regular non-symlink file")
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err = io.Copy(hash, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func contained(root, path string) error {
	info, rootErr := os.Lstat(root)
	if rootErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("owned root must be a real directory")
	}
	if !filepath.IsAbs(path) {
		return errors.New("path must be absolute")
	}
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == "." || !filepath.IsLocal(relative) {
		return errors.New("path escapes owned root")
	}
	current := root
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("symlink in owned path")
		}
	}
	return nil
}

func diskUsage(root string) (int64, uint64, error) {
	var bytes int64
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type().IsRegular() {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			bytes += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, 0, err
	}
	var fs syscall.Statfs_t
	if err = syscall.Statfs(root, &fs); err != nil {
		return 0, 0, err
	}
	return bytes, fs.Bavail * uint64(fs.Bsize), nil
}
