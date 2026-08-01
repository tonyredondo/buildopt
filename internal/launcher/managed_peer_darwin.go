//go:build darwin

package launcher

import (
	"errors"
	"net"
	"os"

	"golang.org/x/sys/unix"
)

func verifyUnixPeerOwner(connection *net.UnixConn) error {
	rawConnection, err := connection.SyscallConn()
	if err != nil {
		return errors.New("inspect managed gateway peer")
	}
	var peer *unix.Xucred
	var credentialErr error
	if err := rawConnection.Control(func(fileDescriptor uintptr) {
		peer, credentialErr = unix.GetsockoptXucred(int(fileDescriptor), unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
	}); err != nil || credentialErr != nil || peer == nil {
		return errors.New("inspect managed gateway peer")
	}
	if peer.Uid != uint32(os.Geteuid()) {
		return errors.New("managed gateway peer has a different user")
	}
	return nil
}
