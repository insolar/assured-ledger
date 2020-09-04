// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build !windows

package l1

import (
	"net"
	"syscall"
)

func ListenTCPWithReuse(network string, laddr *net.TCPAddr) (*net.TCPListener, error) {
	// Linux has SO_REUSEADDR by default for listen operations
	return net.ListenTCP(network, laddr)
}

func DialerTCPWithReuse(laddr *net.TCPAddr) *net.Dialer {
	return &net.Dialer{ LocalAddr: laddr, Control: reuseSocketControl }
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
	})
}
