// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package hostnetwork

import (
	"bytes"
	"context"
	"io"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/instracer"
	"github.com/insolar/assured-ledger/ledger-core/metrics"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/future"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet/types"
	"github.com/insolar/assured-ledger/ledger-core/network/sequence"
)

// NewHostNetwork constructor creates new NewHostNetwork component
func NewHostNetwork(pm uniproto.PeerManager) (network.HostNetwork, error) {
	futureManager := future.NewManager()

	result := &hostNetwork{
		peerManager:       pm,
		handlers:          make(map[types.PacketType]network.RequestHandler),
		sequenceGenerator: sequence.NewGenerator(),
		futureManager:     futureManager,
		responseHandler:   future.NewPacketHandler(futureManager),
	}

	result.streamHandler = NewStreamHandler(result.handleRequest, result.responseHandler)
	return result, nil
}

type hostNetwork struct {
	sequenceGenerator sequence.Generator
	muHandlers        sync.RWMutex
	handlers          map[types.PacketType]network.RequestHandler
	futureManager     future.Manager
	responseHandler   future.PacketHandler
	peerManager       uniproto.PeerManager
	streamHandler     *StreamHandler
}

func (hn *hostNetwork) buildRequest(ctx context.Context, packetType types.PacketType,
	requestData interface{}, receiver nwapi.Address) *rms.Packet {

	result := packet.NewPacket(hn.peerManager.LocalPeer().GetPrimary(), receiver, packetType, uint64(hn.sequenceGenerator.Generate()))
	result.TraceID = inslogger.TraceID(ctx)
	var err error
	result.TraceSpanData, err = instracer.Serialize(ctx)
	if err != nil {
		inslogger.FromContext(ctx).Warn("Network request without span")
	}
	result.SetRequest(requestData)
	return result
}

func (hn *hostNetwork) handleRequest(ctx context.Context, p *packet.ReceivedPacket) {
	logger := inslogger.FromContext(ctx)
	logger.Debugf("Got %s request from host %s; RequestID = %d", p.GetType(), p.Sender.String(), p.RequestID)

	hn.muHandlers.RLock()
	handler, exist := hn.handlers[p.GetType()]
	hn.muHandlers.RUnlock()

	if !exist {
		logger.Warnf("No handler set for packet type %s from node %s", p.GetType(), p.Sender.String())
		ep := hn.BuildResponse(ctx, p, &rms.ErrorResponse{Error: "UNKNOWN RPC ENDPOINT"}).(*rms.Packet)
		ep.RequestID = p.RequestID
		if err := SendPacket(ctx, hn.peerManager, ep); err != nil {
			logger.Errorf("Error while returning error response for request %s from node %s: %s", p.GetType(), p.Sender.String(), err)
		}
		return
	}
	response, err := handler(ctx, p)
	if err != nil {
		logger.Warnf("Error handling request %s from node %s: %s", p.GetType(), p.Sender.String(), err)
		ep := hn.BuildResponse(ctx, p, &rms.ErrorResponse{Error: err.Error()}).(*rms.Packet)
		ep.RequestID = p.RequestID
		if err = SendPacket(ctx, hn.peerManager, ep); err != nil {
			logger.Errorf("Error while returning error response for request %s from node %s: %s", p.GetType(), p.Sender.String(), err)
		}
		return
	}

	if response == nil {
		return
	}

	responsePacket := response.(*rms.Packet)
	responsePacket.RequestID = p.RequestID

	err = SendPacket(ctx, hn.peerManager, responsePacket)
	if err != nil {
		logger.Errorf("Failed to send response: %s", err.Error())
	}
}

// SendRequestToHost send request packet to a remote node.
func (hn *hostNetwork) SendRequestToHost(ctx context.Context, packetType types.PacketType,
	requestData interface{}, receiver nwapi.Address) (network.Future, error) {

	p := hn.buildRequest(ctx, packetType, requestData, receiver)

	inslogger.FromContext(ctx).Debugf("Send %s request to %s with RequestID = %d", p.GetType(), p.Receiver, p.RequestID)

	f := hn.futureManager.Create(p)
	err := SendPacket(ctx, hn.peerManager, p)
	if err != nil {
		f.Cancel()
		return nil, errors.W(err, "Failed to send transport packet")
	}
	metrics.NetworkPacketSentTotal.WithLabelValues(p.GetType().String()).Inc()
	return f, nil
}

// RegisterPacketHandler register a handler function to process incoming request packets of a specific type.
func (hn *hostNetwork) RegisterPacketHandler(t types.PacketType, handler network.RequestHandler) {
	hn.muHandlers.Lock()
	defer hn.muHandlers.Unlock()

	_, exists := hn.handlers[t]
	if exists {
		global.Warnf("Multiple handlers for packet type %s are not supported! New handler will replace the old one!", t)
	}
	hn.handlers[t] = handler
}

// BuildResponse create response to an incoming request with Data set to responseData.
func (hn *hostNetwork) BuildResponse(ctx context.Context, request network.Packet, responseData interface{}) network.Packet {
	result := packet.NewPacket(hn.peerManager.LocalPeer().GetPrimary(), request.GetSenderHost(), request.GetType(), uint64(request.GetRequestID()))
	result.TraceID = inslogger.TraceID(ctx)
	var err error
	result.TraceSpanData, err = instracer.Serialize(ctx)
	if err != nil {
		inslogger.FromContext(ctx).Warn("Network response without span")
	}
	result.SetResponse(responseData)
	return result
}

// RegisterRequestHandler register a handler function to process incoming requests of a specific type.
func (hn *hostNetwork) RegisterRequestHandler(t types.PacketType, handler network.RequestHandler) {
	hn.RegisterPacketHandler(t, handler)
}

func (hn *hostNetwork) ReceiveSmallPacket(packet *uniproto.ReceivedPacket, b []byte) {
	hn.streamHandler.HandleStream(context.Background(), bytes.NewBuffer(b[packet.GetPayloadOffset():len(b)-packet.GetSignatureSize()]))
}

func (hn *hostNetwork) ReceiveLargePacket(rp *uniproto.ReceivedPacket, preRead []byte, r io.LimitedReader) error {
	// fn := rp.NewLargePayloadDeserializer(preRead, r)
	// rp.
	panic("(hn *hostNetwork) ReceiveLargePacket")
	return errors.Unsupported()
}
