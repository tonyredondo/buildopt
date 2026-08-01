//go:build !linux

package launcher

import (
	"errors"
	"net"
)

func newPluginHandshakeListener(string) (net.Listener, string, error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, "", err
	}
	return listener, "tcp://" + listener.Addr().String(), nil
}

func verifyPluginPeer(connection net.Conn) error {
	address, ok := connection.RemoteAddr().(*net.TCPAddr)
	if !ok || address.IP == nil || !address.IP.IsLoopback() {
		return errors.New("plugin peer is not loopback")
	}
	return nil
}
