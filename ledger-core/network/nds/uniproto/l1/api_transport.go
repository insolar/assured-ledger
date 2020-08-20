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

// SessionfulTransportProvider is a meta-factory for session-full connections like TCP and TLS
type SessionfulTransportProvider interface {
	// CreateListeningFactory starts listening and provides OutTransportFactory that has same local address as the listening socket.
	CreateListeningFactory(SessionfulConnectFunc) (OutTransportFactory, error)
	CreateOutgoingOnlyFactory(SessionfulConnectFunc) (OutTransportFactory, error)
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

type SessionlessTransportProvider interface {
	CreateListeningFactory(SessionlessReceiveFunc) (OutTransportFactory, error)
	CreateListeningFactoryWithAddress(SessionlessReceiveFunc, nwapi.Address) (OutTransportFactory, error)
	CreateOutgoingOnlyFactory() (OutTransportFactory, error)
	MaxByteSize() uint16
	Close() error
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
