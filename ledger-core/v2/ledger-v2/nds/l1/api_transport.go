// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l1

import (
	"crypto/x509"
	"io"
	"net"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/ratelimiter"
)

type SessionfulTransport interface {
	Listen(SessionfulConnectFunc) (OutTransportFactory, error)
	Outgoing(SessionfulConnectFunc) (OutTransportFactory, error)
	Close() error
}

type SessionfulConnectFunc func(local, remote Address, conn io.ReadWriteCloser, w OutTransport, err error) (ok bool)

type VerifyPeerCertificateFunc func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error

type OutTransportFactory interface {
	ConnectTo(to Address) (OutTransport, error)
	Close() error
}

/**************************/

type SessionlessTransport interface {
	Listen(SessionlessReceiveFunc) (OutTransportFactory, error)
	Outgoing() (OutTransportFactory, error)
	Close() error
	MaxByteSize() uint16
}

// SessionlessReceiveFunc MUST NOT reuse (b) after return
type SessionlessReceiveFunc func(local, remote Address, b []byte, err error) (ok bool)

type OutTransport interface {
	io.Closer
	Send(payload io.WriterTo) error
	SendBytes(b []byte) error
	GetTag() int
	SetTag(int)

	WithQuota(ratelimiter.RateQuota) OutTransport
}

type OutNetTransport interface {
	io.ReaderFrom
	io.WriteCloser
	OutTransport
	Conn() net.Conn
}
