//go:build windows

package launcher

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/tonyredondo/buildopt/internal/filelock"
	"golang.org/x/sys/windows"
)

const windowsManagedControlStateFile = "gateway-control.json"

type windowsManagedControlState struct {
	SchemaVersion int    `json:"schemaVersion"`
	Address       string `json:"address"`
	Credential    string `json:"credential"`
}

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
	return nil
}

func openPrivateLock(path string) (*os.File, error) {
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, errors.New("managed lock path is a reparse point or not regular")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
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
	command := exec.Command(executable, managedGatewayInternalCommand, config.directory, config.idleTimeout.String())
	command.Stdin, command.Stdout, command.Stderr = nil, nil, nil
	command.Env = []string{}
	command.Dir = filepath.VolumeName(executable) + `\`
	command.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS,
		HideWindow:    true,
	}
	if err := command.Start(); err != nil {
		return fmt.Errorf("start managed gateway process: %w", err)
	}
	return command.Process.Release()
}

func dialManagedGatewayControl(directory string) (net.Conn, string, error) {
	state, err := readWindowsManagedControlState(directory)
	if err != nil {
		return nil, "", err
	}
	connection, err := net.Dial("tcp4", state.Address)
	return connection, state.Credential, err
}

func listenManagedGatewayControl(directory string) (net.Listener, string, error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, "", err
	}
	credentialBytes := make([]byte, 32)
	if _, err := rand.Read(credentialBytes); err != nil {
		_ = listener.Close()
		return nil, "", err
	}
	credential := base64.RawURLEncoding.EncodeToString(credentialBytes)
	state := windowsManagedControlState{1, listener.Addr().String(), credential}
	if err := writeWindowsManagedControlState(directory, state); err != nil {
		_ = listener.Close()
		return nil, "", err
	}
	return listener, credential, nil
}

func verifyManagedGatewayControlPeer(connection net.Conn, expected, supplied string) error {
	remote, remoteOK := connection.RemoteAddr().(*net.TCPAddr)
	local, localOK := connection.LocalAddr().(*net.TCPAddr)
	if !remoteOK || !localOK || !remote.IP.IsLoopback() || !local.IP.IsLoopback() ||
		len(expected) == 0 || len(expected) != len(supplied) ||
		subtle.ConstantTimeCompare([]byte(expected), []byte(supplied)) != 1 {
		return errors.New("managed gateway control authentication failed")
	}
	return nil
}

func readWindowsManagedControlState(directory string) (windowsManagedControlState, error) {
	path := filepath.Join(directory, windowsManagedControlStateFile)
	file, err := openPrivateGatewayState(path)
	if err != nil {
		return windowsManagedControlState{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !privateGatewayFileInfo(info) || info.Size() > managedGatewayMaximumStateBytes {
		return windowsManagedControlState{}, errors.New("managed gateway control state is unsafe")
	}
	raw, err := io.ReadAll(io.LimitReader(file, managedGatewayMaximumStateBytes+1))
	if err != nil || len(raw) > managedGatewayMaximumStateBytes {
		return windowsManagedControlState{}, errors.New("managed gateway control state is unreadable")
	}
	var state windowsManagedControlState
	if err := decodeStrictJSON(raw, &state); err != nil || state.SchemaVersion != 1 ||
		validateManagedGatewayAddress(state.Address) != nil || validateManagedGatewayCredential(state.Credential) != nil {
		return windowsManagedControlState{}, errors.New("managed gateway control state is invalid")
	}
	return state, nil
}

func writeWindowsManagedControlState(directory string, state windowsManagedControlState) error {
	temporary, err := os.CreateTemp(directory, "gateway-control-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := json.NewEncoder(temporary).Encode(state); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return replaceManagedFile(temporaryPath, filepath.Join(directory, windowsManagedControlStateFile))
}

func openPrivateGatewayState(path string) (*os.File, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, errors.New("managed state path is a reparse point or not regular")
	}
	return os.Open(path)
}

func privateGatewayFileInfo(info os.FileInfo) bool {
	return info != nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0
}

func openTrustedManagedFile(path string) (*os.File, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, errors.New("trusted file path is a reparse point or not regular")
	}
	return os.Open(path)
}

func trustedManagedFileInfo(info os.FileInfo) bool {
	return info != nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0
}

func trustedManagedDirectoryInfo(info os.FileInfo) bool {
	return info != nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0
}

func trustedManagedEntryInfo(info os.FileInfo) bool {
	return info != nil && info.Mode()&os.ModeSymlink == 0
}

// Windows read-only authority is enforced by the current-user ACL on the
// package/state roots rather than Unix mode bits, which Go does not expose.
func managedReadOnlyInfo(info os.FileInfo) bool { return info != nil }

func privateManagedDirectoryInfo(info os.FileInfo) bool {
	return info != nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0
}

func gradleDistributionExecutable(root string) string {
	return filepath.Join(root, "bin", "gradle.bat")
}

func validGradleDistributionExecutable(info os.FileInfo) bool {
	return info != nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0
}

func syncManagedDirectory(string) error { return nil }

func replaceManagedFile(source, destination string) error {
	from, err := windows.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(destination)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(
		from,
		to,
		windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH,
	)
}

func managedGatewaySignals() (<-chan os.Signal, func()) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	return signals, func() { signal.Stop(signals) }
}
