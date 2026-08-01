//go:build !windows

package launcher

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/tonyredondo/buildopt/internal/filelock"
	"golang.org/x/sys/unix"
)

func ensurePrivateDirectory(path string, create bool) error {
	if create {
		if err := os.MkdirAll(path, 0o700); err != nil {
			return err
		}
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
	descriptor, err := unix.Open(path, unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(descriptor), path)
	info, err := file.Stat()
	if err != nil || !privateGatewayFileInfo(info) {
		_ = file.Close()
		return nil, errors.New("managed lock is not a private regular file")
	}
	return file, nil
}

func releaseManagedLock(file *os.File) error {
	if file == nil {
		return nil
	}
	return errors.Join(filelock.Unlock(file), file.Close())
}

func startManagedGatewayProcess(config managedGatewayConfig) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve buildopt executable: %w", err)
	}
	null, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer null.Close()
	command := exec.Command(executable, managedGatewayInternalCommand, config.directory, config.idleTimeout.String())
	command.Stdin, command.Stdout, command.Stderr = null, null, null
	command.Env = []string{}
	command.Dir = string(filepath.Separator)
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := command.Start(); err != nil {
		return fmt.Errorf("start managed gateway process: %w", err)
	}
	if err := command.Process.Release(); err != nil {
		_ = command.Process.Kill()
		return err
	}
	return nil
}

func managedGatewayControlAddress(directory string) string {
	digest := sha256.Sum256([]byte(directory))
	return "@buildopt-gateway-" + hex.EncodeToString(digest[:16])
}

func dialManagedGatewayControl(directory string) (net.Conn, string, error) {
	connection, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: managedGatewayControlAddress(directory), Net: "unix"})
	if err != nil {
		return nil, "", err
	}
	if err := verifyUnixPeerOwner(connection); err != nil {
		_ = connection.Close()
		return nil, "", err
	}
	return connection, "", nil
}

func listenManagedGatewayControl(directory string) (net.Listener, string, error) {
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: managedGatewayControlAddress(directory), Net: "unix"})
	return listener, "", err
}

func verifyManagedGatewayControlPeer(connection net.Conn, expected, supplied string) error {
	if expected != "" || supplied != "" {
		return errors.New("unexpected managed gateway control credential")
	}
	unixConnection, ok := connection.(*net.UnixConn)
	if !ok {
		return errors.New("managed gateway control connection is not Unix")
	}
	return verifyUnixPeerOwner(unixConnection)
}

func openPrivateGatewayState(path string) (*os.File, error) {
	descriptor, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(descriptor), path), nil
}

func privateGatewayFileInfo(info os.FileInfo) bool {
	if info == nil {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Geteuid()) && info.Mode().IsRegular() &&
		info.Mode()&os.ModeSymlink == 0 && info.Mode().Perm() == 0o600
}

func openTrustedManagedFile(path string) (*os.File, error) {
	descriptor, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(descriptor), path), nil
}

func trustedManagedFileInfo(info os.FileInfo) bool {
	if info == nil {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && (stat.Uid == uint32(os.Geteuid()) || stat.Uid == 0) &&
		info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0
}

func trustedManagedDirectoryInfo(info os.FileInfo) bool {
	if info == nil {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && (stat.Uid == uint32(os.Geteuid()) || stat.Uid == 0) &&
		info.IsDir() && info.Mode()&os.ModeSymlink == 0
}

func trustedManagedEntryInfo(info os.FileInfo) bool {
	if info == nil {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && (stat.Uid == uint32(os.Geteuid()) || stat.Uid == 0) &&
		info.Mode()&os.ModeSymlink == 0
}

func managedReadOnlyInfo(info os.FileInfo) bool {
	return info != nil && info.Mode().Perm()&0o222 == 0
}

func privateManagedDirectoryInfo(info os.FileInfo) bool {
	if info == nil {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Geteuid()) && info.IsDir() &&
		info.Mode()&os.ModeSymlink == 0 && info.Mode().Perm() == 0o700
}

func gradleDistributionExecutable(root string) string {
	return filepath.Join(root, "bin", "gradle")
}

func validGradleDistributionExecutable(info os.FileInfo) bool {
	return info != nil && info.Mode().IsRegular() &&
		info.Mode()&os.ModeSymlink == 0 && info.Mode().Perm()&0o100 != 0
}

func syncManagedDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func replaceManagedFile(source, destination string) error {
	return os.Rename(source, destination)
}

func managedGatewaySignals() (<-chan os.Signal, func()) {
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	return signals, func() { signal.Stop(signals) }
}
