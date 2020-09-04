// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build !windows

package l1

import (
	"context"
	"net"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func ListenTCPWithReuse(network string, laddr *net.TCPAddr) (*net.TCPListener, error) {
	sl := &net.ListenConfig{Control: reuseSocketControl}
	listener, err := sl.Listen(context.Background(), network, laddr.String())
	if err != nil {
		return nil, err
	}
	return listener.(*net.TCPListener), nil
}

func DialerTCPWithReuse(laddr *net.TCPAddr) *net.Dialer {
	return &net.Dialer{LocalAddr: laddr, Control: reuseSocketControl}
}

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
		err = os.NewSyscallError("setsockopt", syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1))
		if err != nil {
			return
		}
		err = os.NewSyscallError("setsockopt", syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, unix.SO_REUSEPORT, 1))
	})
}
