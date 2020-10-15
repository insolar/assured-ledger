// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package hostnetwork

import (
	"context"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/iokit"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/instracer"
	"github.com/insolar/assured-ledger/ledger-core/metrics"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/future"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet"
)

// RequestHandler is callback function for request handling
type RequestHandler func(ctx context.Context, p *packet.ReceivedPacket)

// StreamHandler parses packets from data stream and calls request handler or response handler
type StreamHandler struct {
	requestHandler  RequestHandler
	responseHandler future.PacketHandler
}

// NewStreamHandler creates new StreamHandler
func NewStreamHandler(requestHandler RequestHandler, responseHandler future.PacketHandler) *StreamHandler {
	return &StreamHandler{
		requestHandler:  requestHandler,
		responseHandler: responseHandler,
	}
}

func (s *StreamHandler) HandleStream(ctx context.Context, reader io.Reader) {
	mainLogger := inslogger.FromContext(ctx)

	logLevel := inslogger.GetLoggerLevel(ctx)
	// get only log level from context, discard TraceID in favor of packet TraceID
	packetCtx := inslogger.WithLoggerLevel(context.Background(), logLevel)

	p, length, err := packet.DeserializePacket(mainLogger, reader)
	if err != nil {
		if network.IsConnectionClosed(err) || network.IsClosedPipe(err) {
			mainLogger.Info("[ HandleStream ] Connection closed.")
			return
		}

		mainLogger.Warnf("[ HandleStream ] Failed to deserialize packet: ", err.Error())
		panic("[ HandleStream ] Failed to deserialize packet")
		return
	}

	packetCtx, logger := inslogger.WithTraceField(packetCtx, p.TraceID)
	span, err := instracer.Deserialize(p.TraceSpanData)
	if err == nil {
		packetCtx = instracer.WithParentSpan(packetCtx, span)
	} else {
		inslogger.FromContext(packetCtx).Warn("Incoming packet without span")
	}
	logger.Debugf("[ HandleStream ] Handling packet RequestID = %d, size = %d", p.RequestID, length)
	metrics.NetworkRecvSize.Observe(float64(length))
	if p.IsResponse() {
		go s.responseHandler.Handle(packetCtx, p)
	} else {
		go s.requestHandler(packetCtx, p)
	}
}

// SendPacket sends packet using connection from pool
func SendPacket(ctx context.Context, pm uniproto.PeerManager, p *rms.Packet) error {
	data, err := packet.SerializePacket(p)
	if err != nil {
		return errors.W(err, "Failed to serialize packet")
	}

	if p.Receiver.Port() == 0 || !p.Receiver.CanConnect() {
		panic("invalid address")
	}

	var peer uniproto.Peer
	peer, err = pm.ConnectedPeer(p.Receiver)
	if err != nil {
		peer, err = pm.ConnectPeer(p.Receiver)
		if err != nil {
			return errors.W(err, "Failed to get connection")
		}
	}

	preparedPacket := &BootstrapPacket{Payload: data}
	err = peer.SendPacket(uniproto.SessionfulSmall, preparedPacket)
	if err != nil {
		// retry
		inslogger.FromContext(ctx).Warn("[ SendPacket ] retry conn.Write")
		err = peer.SendPacket(uniproto.SessionfulSmall, preparedPacket)
		if err != nil {
			return errors.W(err, "[ SendPacket ] Failed to write data")
		}
	}

	metrics.NetworkSentSize.Observe(float64(len(data)))
	return nil
}

var _ uniproto.ProtocolPacket = &BootstrapPacket{}

type BootstrapPacket struct {
	Payload []byte
}

func (p *BootstrapPacket) PreparePacket() (uniproto.PacketTemplate, uint, uniproto.PayloadSerializerFunc) {
	pt := uniproto.PacketTemplate{}
	pt.Header.SetRelayRestricted(true)
	pt.Header.SetProtocolType(uniproto.ProtocolTypeJoinCandidate)
	pt.PulseNumber = pulse.MinTimePulse
	return pt, uint(len(p.Payload)), p.SerializePayload
}

func (p *BootstrapPacket) SerializePayload(_ nwapi.SerializationContext, _ *uniproto.Packet, writer *iokit.LimitedWriter) error {
	_, err := writer.Write(p.Payload)
	return err
}

func (p *BootstrapPacket) DeserializePayload(_ nwapi.DeserializationContext, _ *uniproto.Packet, reader *iokit.LimitedReader) error {
	b := make([]byte, reader.RemainingBytes())
	_, err := io.ReadFull(reader, b)
	p.Payload = b
	return err
}
