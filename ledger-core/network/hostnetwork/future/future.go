package future

import (
	"sync/atomic"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/metrics"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet/types"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/legacyhost"
)

type future struct {
	response       chan network.ReceivedPacket
	receiver       *legacyhost.Host
	request        *rms.Packet
	requestID      types.RequestID
	cancelCallback CancelCallback
	finished       uint32
}

// NewFuture creates a new Future.
func NewFuture(requestID types.RequestID, receiver *legacyhost.Host, packet *rms.Packet, cancelCallback CancelCallback) Future {
	metrics.NetworkFutures.WithLabelValues(packet.GetType().String()).Inc()
	return &future{
		response:       make(chan network.ReceivedPacket, 1),
		receiver:       receiver,
		request:        packet,
		requestID:      requestID,
		cancelCallback: cancelCallback,
	}
}

// ID returns RequestID of packet.
func (f *future) ID() types.RequestID {
	return f.requestID
}

// Receiver returns Host address that was used to create packet.
func (f *future) Receiver() *legacyhost.Host {
	return f.receiver
}

// Request returns original request packet.
func (f *future) Request() network.Packet {
	return f.request
}

// Response returns response packet channel.
func (f *future) Response() <-chan network.ReceivedPacket {
	return f.response
}

// SetResponse write packet to the response channel.
func (f *future) SetResponse(response network.ReceivedPacket) {
	if atomic.CompareAndSwapUint32(&f.finished, 0, 1) {
		f.response <- response
		f.finish()
	}
}

// WaitResponse gets the future response from Response() channel with a timeout set to `duration`.
func (f *future) WaitResponse(duration time.Duration) (network.ReceivedPacket, error) {
	select {
	case response, ok := <-f.Response():
		if !ok {
			return nil, ErrChannelClosed
		}
		return response, nil
	case <-time.After(duration):
		f.Cancel()
		metrics.NetworkPacketTimeoutTotal.WithLabelValues(f.request.GetType().String()).Inc()
		return nil, ErrTimeout
	}
}

// Cancel cancels Future processing.
// Please note that cancelCallback is called asynchronously. In other words it's not guaranteed
// it was called and finished when WaitResponse() returns ErrChannelClosed.
func (f *future) Cancel() {
	if atomic.CompareAndSwapUint32(&f.finished, 0, 1) {
		f.finish()
	}
}

func (f *future) finish() {
	close(f.response)
	f.cancelCallback(f)
}
