//go:build linux

package launcher

import (
	"errors"
	"net"
	"os"
	"syscall"
)

func verifyUnixPeerOwner(connection *net.UnixConn) error {
	rawConnection, err := connection.SyscallConn()
	if err != nil {
		return errors.New("inspect managed gateway peer")
	}
	var peer *syscall.Ucred
	var credentialErr error
	if err := rawConnection.Control(func(fileDescriptor uintptr) {
		peer, credentialErr = syscall.GetsockoptUcred(int(fileDescriptor), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	}); err != nil || credentialErr != nil || peer == nil {
		return errors.New("inspect managed gateway peer")
	}
	if peer.Uid != uint32(os.Geteuid()) {
		return errors.New("managed gateway peer has a different user")
	}
	return nil
}
