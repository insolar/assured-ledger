// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l1

import (
	"crypto/x509"
	"io"
	"net"

	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/ratelimiter"
)

type SessionfulTransport interface {
	Listen(SessionfulConnectFunc) (OutTransportFactory, error)
	Outgoing(SessionfulConnectFunc) (OutTransportFactory, error)
	Close() error
}

type SessionfulConnectFunc func(local, remote nwapi.Address, conn io.ReadWriteCloser, w OutTransport, err error) (ok bool)

type VerifyPeerCertificateFunc func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error

type OutTransportFactory interface {
	LocalAddr() nwapi.Address
	ConnectTo(nwapi.Address, nwapi.Preference) (OutTransport, error)
	Close() error
}

/**************************/

type SessionlessTransport interface {
	Listen(SessionlessReceiveFunc) (OutTransportFactory, error)
	ListenOverride(SessionlessReceiveFunc, nwapi.Address) (OutTransportFactory, error)
	Outgoing() (OutTransportFactory, error)
	Close() error
	MaxByteSize() uint16
}

// SessionlessReceiveFunc MUST NOT reuse (b) after return
type SessionlessReceiveFunc func(local, remote nwapi.Address, b []byte, err error) (ok bool)

type BasicOutTransport interface {
	Send(payload io.WriterTo) error
	SendBytes(b []byte) error
}

type OutTransport interface {
	BasicOutTransport
	io.Closer
	GetTag() int
	SetTag(int)

	WithQuota(ratelimiter.RateQuota) OutTransport
}

type TwoWayTransport interface {
	OutTransport
	TwoWayConn() io.ReadWriteCloser
}

type SemiTransport interface {
	TwoWayTransport
	// ConnectReceiver with (nil) arg will use receive func of parent transport
	ConnectReceiver(SessionfulConnectFunc) (bool, TwoWayTransport)
}

type OutNetTransport interface {
	io.ReaderFrom
	io.Writer
	OutTransport
	NetConn() net.Conn
}
