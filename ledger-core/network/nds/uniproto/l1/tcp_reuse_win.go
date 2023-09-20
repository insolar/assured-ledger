// +build windows

package l1

import (
	"context"
	"net"
	"os"
	"syscall"
)

//nolint:interfacer
func ListenTCPWithReuse(network string, laddr *net.TCPAddr) (*net.TCPListener, error) {
	sl := &net.ListenConfig{ Control: reuseSocketControl }
	listener, err := sl.Listen(context.Background(), network, laddr.String())
	if err != nil {
		return nil, err
	}
	return listener.(*net.TCPListener), nil
}

//nolint:interfacer
func DialerTCPWithReuse(laddr *net.TCPAddr) *net.Dialer {
	return &net.Dialer{ LocalAddr: laddr, Control: reuseSocketControl }
}

//nolint:interfacer
func DialTCPWithReuse(network string, laddr, raddr *net.TCPAddr) (*net.TCPConn, error) {
	sl := DialerTCPWithReuse(laddr)
	conn, err := sl.Dial(network, raddr.String())
	if err != nil {
		return nil, err
	}
	return conn.(*net.TCPConn), nil
}

func reuseSocketControl(_ string, _ string, c syscall.RawConn) (err error) {
	return c.Control(func(fd uintptr) {
		err = os.NewSyscallError("setsockopt", syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1))
	})
}
