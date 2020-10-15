// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package future

import (
	"context"
	"errors"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet/types"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

var (
	// ErrTimeout is returned when the operation timeout is exceeded.
	ErrTimeout = errors.New("timeout")
	// ErrChannelClosed is returned when the input channel is closed.
	ErrChannelClosed = errors.New("channel closed")
)

// Future is network response future.
type Future interface {

	// ID returns packet sequence number.
	ID() types.RequestID

	// Receiver returns the initiator of the packet.
	Receiver() nwapi.Address

	// Request returns origin request.
	Request() network.Packet

	// Response is a channel to listen for future response.
	Response() <-chan network.ReceivedPacket

	// SetResponse makes packet to appear in response channel.
	SetResponse(network.ReceivedPacket)

	// WaitResponse gets the future response from Response() channel with a timeout set to `duration`.
	WaitResponse(duration time.Duration) (network.ReceivedPacket, error)

	// Cancel closes all channels and cleans up underlying structures.
	Cancel()
}

// CancelCallback is a callback function executed when cancelling Future.
type CancelCallback func(Future)

type Manager interface {
	Get(packet *rms.Packet) Future
	Create(packet *rms.Packet) Future
}

type PacketHandler interface {
	Handle(ctx context.Context, msg *packet.ReceivedPacket)
}
