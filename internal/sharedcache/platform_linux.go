//go:build linux

package sharedcache

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

const (
	afsSuperMagic   = 0x5346414f
	bcachefsMagic   = 0xca451a4e
	btrfsSuperMagic = 0x9123683e
	cephSuperMagic  = 0x00c36400
	cifsSuperMagic  = 0xff534d42
	codaSuperMagic  = 0x73757245
	ecryptfsMagic   = 0x0000f15f
	extSuperMagic   = 0x0000ef53
	f2fsSuperMagic  = 0xf2f52010
	fuseSuperMagic  = 0x65735546
	gfs2SuperMagic  = 0x01161970
	ncpSuperMagic   = 0x0000564c
	nfsSuperMagic   = 0x00006969
	nilfsSuperMagic = 0x00003434
	ocfs2SuperMagic = 0x7461636f
	overlayfsMagic  = 0x794c7630
	ramfsMagic      = 0x858458f6
	reiserfsMagic   = 0x52654973
	smbSuperMagic   = 0x0000517b
	tmpfsMagic      = 0x01021994
	ubifsMagic      = 0x24051905
	v9fsSuperMagic  = 0x01021997
	xfsSuperMagic   = 0x58465342
	zfsSuperMagic   = 0x2fc12fc1
)

func preparePrivateDirectory(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is not a private directory", path)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) {
		return fmt.Errorf("%s is not owned by the current user", path)
	}
	if info.Mode().Perm() != 0o700 {
		return fmt.Errorf("%s must have mode 0700", path)
	}
	return nil
}

func openPrivateLock(path string) (*os.File, error) {
	return openPrivateFile(path, unix.O_CREAT|unix.O_RDWR, 0o600)
}

func preparePrivateDatabase(path string) error {
	file, err := openPrivateFile(
		path,
		unix.O_CREAT|unix.O_RDWR,
		0o600,
	)
	if err != nil {
		return err
	}
	return file.Close()
}

func openPrivateBlob(path string) (*os.File, error) {
	return openPrivateFile(path, unix.O_RDONLY, 0)
}

func openPrivateFile(path string, flags int, mode uint32) (*os.File, error) {
	fd, err := unix.Open(
		path,
		flags|unix.O_CLOEXEC|unix.O_NOFOLLOW,
		mode,
	)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = unix.Close(fd)
		return nil, errors.New("create private file handle")
	}
	if err := validatePrivateRegularFile(file); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return file, nil
}

func validatePrivateRegularFile(file *os.File) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) {
		return errors.New("file is not owned by the current user")
	}
	if !info.Mode().IsRegular() ||
		info.Mode()&os.ModeSymlink != 0 ||
		info.Mode().Perm() != 0o600 ||
		stat.Nlink != 1 {
		return errors.New(
			"file must be regular, singly linked, and mode 0600",
		)
	}
	return nil
}

func validatePrivateSidecar(path string) error {
	file, err := openPrivateFile(path, unix.O_RDONLY, 0)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return file.Close()
}

func acquireExclusiveLock(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
}

func releaseExclusiveLock(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_UN)
}

func isLockBusy(err error) bool {
	return errors.Is(err, unix.EWOULDBLOCK) ||
		errors.Is(err, unix.EAGAIN)
}

func validateLocalStorageFilesystem(paths ...string) error {
	if len(paths) == 0 {
		return errors.New("no storage paths")
	}
	var expectedDevice uint64
	for index, path := range paths {
		var statfs unix.Statfs_t
		if err := unix.Statfs(path, &statfs); err != nil {
			return fmt.Errorf("statfs %s: %w", path, err)
		}
		if !isSupportedLocalFilesystemType(int64(statfs.Type)) {
			return fmt.Errorf(
				"%s uses filesystem type %#x, which is not proven local",
				path,
				statfs.Type,
			)
		}
		var stat unix.Stat_t
		if err := unix.Stat(path, &stat); err != nil {
			return fmt.Errorf("stat %s: %w", path, err)
		}
		if index == 0 {
			expectedDevice = uint64(stat.Dev)
			continue
		}
		if uint64(stat.Dev) != expectedDevice {
			return fmt.Errorf("%s is on a different filesystem", path)
		}
	}
	return nil
}

func isSupportedLocalFilesystemType(filesystemType int64) bool {
	switch filesystemType {
	case bcachefsMagic,
		btrfsSuperMagic,
		ecryptfsMagic,
		extSuperMagic,
		f2fsSuperMagic,
		nilfsSuperMagic,
		overlayfsMagic,
		ramfsMagic,
		reiserfsMagic,
		tmpfsMagic,
		ubifsMagic,
		xfsSuperMagic,
		zfsSuperMagic:
		return true
	default:
		return false
	}
}

func syncDirectory(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return file.Sync()
}
