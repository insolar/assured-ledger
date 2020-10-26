// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package hostnetwork

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/insolar/assured-ledger/ledger-core/rms"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/instracer"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/metrics"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/future"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet/types"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/pool"
	"github.com/insolar/assured-ledger/ledger-core/network/sequence"
	"github.com/insolar/assured-ledger/ledger-core/network/transport"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms/legacyhost"
)

// NewHostNetwork constructor creates new NewHostNetwork component
func NewHostNetwork(nodeRef string) (network.HostNetwork, error) {
	id, err := reference.GlobalFromString(nodeRef)
	if err != nil {
		return nil, errors.W(err, "invalid nodeRef")
	}

	futureManager := future.NewManager()

	result := &hostNetwork{
		handlers:          make(map[types.PacketType]network.RequestHandler),
		sequenceGenerator: sequence.NewGenerator(),
		nodeID:            id,
		futureManager:     futureManager,
		responseHandler:   future.NewPacketHandler(futureManager),
	}

	return result, nil
}

func NewNetwork(host legacyhost.Host) (network.HostNetwork, error) {
	futureManager := future.NewManager()

	result := &hostNetwork{
		handlers:          make(map[types.PacketType]network.RequestHandler),
		sequenceGenerator: sequence.NewGenerator(),
		nodeID:            host.NodeID,
		shortId:           host.ShortID,
		futureManager:     futureManager,
		responseHandler:   future.NewPacketHandler(futureManager),
	}

	return result, nil
}

type hostNetwork struct {
	Resolver network.RoutingTable `inject:""`
	Factory  transport.Factory    `inject:""`

	nodeID            reference.Global
	shortId           node.ShortNodeID
	started           uint32
	transport         transport.StreamTransport
	sequenceGenerator sequence.Generator
	muHandlers        sync.RWMutex
	handlers          map[types.PacketType]network.RequestHandler
	futureManager     future.Manager
	responseHandler   future.PacketHandler
	pool              pool.ConnectionPool

	muOrigin sync.RWMutex
	origin   *legacyhost.Host
}

// Start listening to network requests, should be started in goroutine.
func (hn *hostNetwork) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapUint32(&hn.started, 0, 1) {
		inslogger.FromContext(ctx).Warn("HostNetwork component already started")
		return nil
	}

	handler := NewStreamHandler(hn.handleRequest, hn.responseHandler)

	var err error
	hn.transport, err = hn.Factory.CreateStreamTransport(handler)
	if err != nil {
		return errors.W(err, "Failed to create stream transport")
	}

	hn.pool = pool.NewConnectionPool(hn.transport)

	hn.muOrigin.Lock()
	defer hn.muOrigin.Unlock()

	if err := hn.transport.Start(ctx); err != nil {
		return errors.W(err, "failed to start stream transport")
	}

	h, err := legacyhost.NewHostNS(hn.transport.Address(), hn.nodeID, hn.shortId)
	if err != nil {
		return errors.W(err, "failed to create host")
	}

	hn.origin = h

	return nil
}

// Stop listening to network requests.
func (hn *hostNetwork) Stop(ctx context.Context) error {
	if atomic.CompareAndSwapUint32(&hn.started, 1, 0) {
		hn.pool.Reset()
		err := hn.transport.Stop(ctx)
		if err != nil {
			return errors.W(err, "Failed to stop transport.")
		}
	}
	return nil
}

func (hn *hostNetwork) buildRequest(ctx context.Context, packetType types.PacketType,
	requestData interface{}, receiver *legacyhost.Host) *rms.Packet {

	result := packet.NewPacket(hn.getOrigin(), receiver, packetType, uint64(hn.sequenceGenerator.Generate()))
	result.TraceID = inslogger.TraceID(ctx)
	var err error
	result.TraceSpanData, err = instracer.Serialize(ctx)
	if err != nil {
		inslogger.FromContext(ctx).Warn("Network request without span")
	}
	result.SetRequest(requestData)
	return result
}

// PublicAddress returns public address that can be published for all nodes.
func (hn *hostNetwork) PublicAddress() string {
	return hn.getOrigin().Address.String()
}

func (hn *hostNetwork) handleRequest(ctx context.Context, p *packet.ReceivedPacket) {
	logger := inslogger.FromContext(ctx)
	logger.Debugf("Got %s request from host %s; RequestID = %d", p.GetType(), p.Sender, p.RequestID)

	hn.muHandlers.RLock()
	handler, exist := hn.handlers[p.GetType()]
	hn.muHandlers.RUnlock()

	if !exist {
		logger.Warnf("No handler set for packet type %s from node %s", p.GetType(), p.Sender.NodeID)
		ep := hn.BuildResponse(ctx, p, &rms.ErrorResponse{Error: "UNKNOWN RPC ENDPOINT"}).(*rms.Packet)
		ep.RequestID = p.RequestID
		if err := SendPacket(ctx, hn.pool, ep); err != nil {
			logger.Errorf("Error while returning error response for request %s from node %s: %s", p.GetType(), p.Sender.NodeID, err)
		}
		return
	}
	response, err := handler(ctx, p)
	if err != nil {
		logger.Warnf("Error handling request %s from node %s: %s", p.GetType(), p.Sender.NodeID, err)
		ep := hn.BuildResponse(ctx, p, &rms.ErrorResponse{Error: err.Error()}).(*rms.Packet)
		ep.RequestID = p.RequestID
		if err = SendPacket(ctx, hn.pool, ep); err != nil {
			logger.Errorf("Error while returning error response for request %s from node %s: %s", p.GetType(), p.Sender.NodeID, err)
		}
		return
	}

	if response == nil {
		return
	}

	responsePacket := response.(*rms.Packet)
	responsePacket.RequestID = p.RequestID
	err = SendPacket(ctx, hn.pool, responsePacket)
	if err != nil {
		logger.Errorf("Failed to send response: %s", err.Error())
	}
}

// SendRequestToHost send request packet to a remote node.
func (hn *hostNetwork) SendRequestToHost(ctx context.Context, packetType types.PacketType,
	requestData interface{}, receiver *legacyhost.Host) (network.Future, error) {

	if atomic.LoadUint32(&hn.started) == 0 {
		return nil, errors.New("host network is not started")
	}

	p := hn.buildRequest(ctx, packetType, requestData, receiver)

	inslogger.FromContext(ctx).Debugf("Send %s request to %s with RequestID = %d", p.GetType(), p.Receiver, p.RequestID)

	f := hn.futureManager.Create(p)
	err := SendPacket(ctx, hn.pool, p)
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
	result := packet.NewPacket(hn.getOrigin(), request.GetSenderHost(), request.GetType(), uint64(request.GetRequestID()))
	result.TraceID = inslogger.TraceID(ctx)
	var err error
	result.TraceSpanData, err = instracer.Serialize(ctx)
	if err != nil {
		inslogger.FromContext(ctx).Warn("Network response without span")
	}
	result.SetResponse(responseData)
	return result
}

// SendRequest send request to a remote node.
func (hn *hostNetwork) SendRequest(ctx context.Context, packetType types.PacketType,
	requestData interface{}, receiver reference.Global) (network.Future, error) {

	h, err := hn.Resolver.Resolve(receiver)
	if err != nil {
		return nil, errors.W(err, "error resolving NodeID -> Address")
	}
	return hn.SendRequestToHost(ctx, packetType, requestData, h)
}

// RegisterRequestHandler register a handler function to process incoming requests of a specific type.
func (hn *hostNetwork) RegisterRequestHandler(t types.PacketType, handler network.RequestHandler) {
	hn.RegisterPacketHandler(t, handler)
}

func (hn *hostNetwork) getOrigin() *legacyhost.Host {
	hn.muOrigin.RLock()
	defer hn.muOrigin.RUnlock()

	return hn.origin
}
