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

// SessionfulTransportProvider is a meta-factory for session-full, bi-directional connections like TCP and TLS.
type SessionfulTransportProvider interface {
	// CreateListeningFactory starts listening and provides OutTransportFactory that has same local address as the listening socket.
	// Param (SessionfulConnectFunc) is to handle either inbound connection(s) from listen or inbound direction of an outgoing connection.
	CreateListeningFactory(SessionfulConnectFunc) (OutTransportFactory, nwapi.Address, error)
	// CreateOutgoingOnlyFactory creates a factory to open outgoing connections.
	// Param (SessionfulConnectFunc) is to handle inbound flow of an outgoing connection.
	CreateOutgoingOnlyFactory(SessionfulConnectFunc) (OutTransportFactory, error)
	Close() error
}

// SessionfulConnectFunc receives either inbound connection(s) from listen or inbound direction of an outgoing connection.
// Param (w) is not nil for an inbound connection (for an outbound connection it is returned by ConnectTo).
// Param (conn) is a raw bi-directional connection.
type SessionfulConnectFunc func(local, remote nwapi.Address, conn io.ReadWriteCloser, w OutTransport, err error) (ok bool)

// VerifyPeerCertificateFunc verifies raw and pre-verified certs from a connection. Errors should be returned when credentials aren't allowed to connect.
type VerifyPeerCertificateFunc func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error

// OutTransportFactory represents a generic factory to create/open an outgoing connection.
type OutTransportFactory interface {
	// ConnectTo resolves provided address and makes a connection
	// Can return (nil, nil) when the given address is of unsupported type.
	ConnectTo(nwapi.Address) (OutTransport, error)
	Close() error
}

/**************************/

// SessionlessTransportProvider is a meta-factory for session-less, one-way connections like UDP
type SessionlessTransportProvider interface {
	// CreateListeningFactory starts listening and provides OutTransportFactory that has same local address as the listening socket.
	// Param (SessionlessReceiveFunc) is to handle inbound datagrams.
	CreateListeningFactory(SessionlessReceiveFunc) (OutTransportFactory, nwapi.Address, error)
	// CreateListeningFactoryWithAddress is similar to CreateListeningFactory but uses the given address. Provided address must have port.
	CreateListeningFactoryWithAddress(SessionlessReceiveFunc, nwapi.Address) (OutTransportFactory, error)
	// CreateOutgoingOnlyFactory creates a factory to open outgoing connections.
	CreateOutgoingOnlyFactory() (OutTransportFactory, error)
	// MaxByteSize returns a maximum size of a supported/deliverable datagram.
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

	// TwoWayConn returns an underlying two-way connection. Only applies to a sessionful transport.
	TwoWayConn() io.ReadWriteCloser
}

// SemiTransport represents a bi-directional transport without a receiving logic attached yet.
type SemiTransport interface {
	TwoWayTransport

	// ConnectReceiver can be used to connect receive side of this transport. It can only be called once.
	// When param is (nil) it will use a receiver func of its factory.
	// Returns result of the applied SessionfulConnectFunc and a relevant transport.
	// Will return (false, nil) when operation is not available, e.g. on repeated call.
	ConnectReceiver(SessionfulConnectFunc) (bool, TwoWayTransport)
}

type OutNetTransport interface {
	io.ReaderFrom
	io.Writer
	OutTransport
	NetConn() net.Conn
}
