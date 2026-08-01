//go:build linux

package launcher

import (
	"errors"
	"net"
	"os"
	"syscall"
)

func newPluginHandshakeListener(socketPath string) (net.Listener, string, error) {
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: socketPath, Net: "unix"})
	if err != nil {
		return nil, "", err
	}
	return listener, socketPath, nil
}

func verifyPluginPeer(connection net.Conn) error {
	unixConnection, ok := connection.(*net.UnixConn)
	if !ok {
		return errors.New("plugin peer is not a Unix socket")
	}
	rawConnection, err := unixConnection.SyscallConn()
	if err != nil {
		return errors.New("inspect plugin peer")
	}
	var peer *syscall.Ucred
	var credentialErr error
	if err := rawConnection.Control(func(fileDescriptor uintptr) {
		peer, credentialErr = syscall.GetsockoptUcred(int(fileDescriptor), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	}); err != nil || credentialErr != nil || peer == nil {
		return errors.New("inspect plugin peer")
	}
	if peer.Uid != uint32(os.Geteuid()) {
		return errors.New("plugin peer user does not own the launcher")
	}
	return nil
}
