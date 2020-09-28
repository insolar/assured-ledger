// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package adapters

import (
	"bytes"
	"context"
	"io"
	"sync"
	"sync/atomic"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/core/errors"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insmetrics"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/common/warning"

	"go.opencensus.io/stats"

	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/common/endpoints"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/transport"
)

type PacketProcessor interface {
	ProcessPacket(ctx context.Context, payload transport.PacketParser, from endpoints.Inbound) error
}

type PacketParserFactory interface {
	ParsePacket(ctx context.Context, reader io.Reader) (transport.PacketParser, error)
}

type packetHandler struct {
	packetProcessor PacketProcessor
}

func newPacketHandler(packetProcessor PacketProcessor) *packetHandler {
	return &packetHandler{
		packetProcessor: packetProcessor,
	}
}

func (ph *packetHandler) handlePacket(ctx context.Context, packetParser transport.PacketParser, sender string) {
	ctx, logger := PacketLateLogger(ctx, packetParser)

	if logger.Is(log.DebugLevel) {
		logger.Debugf("Received packet %v", packetParser)
	}

	err := ph.packetProcessor.ProcessPacket(ctx, packetParser, &endpoints.InboundConnection{
		Addr: endpoints.Name(sender),
	})

	if err == nil {
		return
	}

	switch err.(type) {
	case warning.Warning:
		break
	default:
		// Temporary hide pulse number mismatch error https://insolar.atlassian.net/browse/INS-3943
		if mismatch, _ := errors.IsMismatchPulseError(err); mismatch {
			break
		}

		logger.Error("Failed to process packet: ", err)
	}

	logger.Warn("Failed to process packet: ", err)
}

type DatagramHandler struct {
	mu                  sync.RWMutex
	inited              uint32
	packetHandler       *packetHandler
	packetParserFactory PacketParserFactory
}

func NewDatagramHandler() *DatagramHandler {
	return &DatagramHandler{}
}

func (dh *DatagramHandler) SetPacketProcessor(packetProcessor PacketProcessor) {
	dh.mu.Lock()
	defer dh.mu.Unlock()

	dh.packetHandler = newPacketHandler(packetProcessor)
}

func (dh *DatagramHandler) SetPacketParserFactory(packetParserFactory PacketParserFactory) {
	dh.mu.Lock()
	defer dh.mu.Unlock()

	dh.packetParserFactory = packetParserFactory
}

func (dh *DatagramHandler) isInitialized(ctx context.Context) bool {
	if atomic.LoadUint32(&dh.inited) == 0 {
		dh.mu.RLock()
		defer dh.mu.RUnlock()

		if dh.packetHandler == nil {
			inslogger.FromContext(ctx).Error("Packet handler is not initialized")
			return false
		}

		if dh.packetParserFactory == nil {
			inslogger.FromContext(ctx).Error("Packet parser factory is not initialized")
			return false
		}
		atomic.StoreUint32(&dh.inited, 1)
	}
	return true
}

func (dh *DatagramHandler) HandleDatagram(ctx context.Context, address string, buf []byte) {
	// todo replace with new

	ctx, logger := PacketEarlyLogger(ctx, address)

	if !dh.isInitialized(ctx) {
		return
	}

	packetParser, err := dh.packetParserFactory.ParsePacket(ctx, bytes.NewReader(buf))
	if err != nil {
		stats.Record(ctx, network.ConsensusPacketsRecvBad.M(int64(len(buf))))
		logger.Warnf("Failed to get PacketParser: ", err)
		return
	}

	ctx = insmetrics.InsertTag(ctx, network.TagPhase, packetParser.GetPacketType().String())
	stats.Record(ctx, network.ConsensusPacketsRecv.M(int64(len(buf))))

	dh.packetHandler.handlePacket(ctx, packetParser, address)
}

var _ uniproto.Controller = &ConsensusProtocolMarshaller{}
var _ uniproto.Receiver = &ConsensusProtocolMarshaller{}

type ConsensusProtocolMarshaller struct {
	HandlerAdapter *DatagramHandler

	LastFrom   nwapi.Address
	LastPacket uniproto.Packet
	LastBytes  []byte
	LastMsg    string
	LastSigLen int
	LastLarge  bool
	LastError  error
	ReportErr  error
}

func (p *ConsensusProtocolMarshaller) Start(manager uniproto.PeerManager) {}
func (p *ConsensusProtocolMarshaller) NextPulse(p2 pulse.Range)           {}
func (p *ConsensusProtocolMarshaller) Stop()                              {}

func (p *ConsensusProtocolMarshaller) PrepareHeader(_ *uniproto.Header, pn pulse.Number) (pulse.Number, error) {
	return pn, nil
}

func (p *ConsensusProtocolMarshaller) VerifyHeader(*uniproto.Header, pulse.Number) error {
	return nil
}

func (p *ConsensusProtocolMarshaller) ReceiveSmallPacket(packet *uniproto.ReceivedPacket, b []byte) {
	// todo: forward packet

	p.LastFrom = packet.From
	p.LastPacket = packet.Packet
	p.LastBytes = append([]byte(nil), b...)
	p.LastSigLen = packet.GetSignatureSize()
	p.LastLarge = false

	p.HandlerAdapter.HandleDatagram(context.Background(), packet.From.String(), b[packet.GetPayloadOffset():len(b)-int(p.LastSigLen)])

	// p.LastMsg = string(b[packet.GetPayloadOffset() : len(b)-int(p.LastSigLen)])
	//
	// p.LastError = packet.NewSmallPayloadDeserializer(b)(nil, func(_ nwapi.DeserializationContext, packet *uniproto.Packet, reader *iokit.LimitedReader) error {
	// 	b := make([]byte, reader.RemainingBytes())
	// 	n, err := io.ReadFull(reader, b)
	// 	p.LastMsg = string(b[:n])
	// 	return err
	// })

	return
}

func (p *ConsensusProtocolMarshaller) ReceiveLargePacket(packet *uniproto.ReceivedPacket, preRead []byte, r io.LimitedReader) error {
	return throw.Unsupported()
}

func (p *ConsensusProtocolMarshaller) SerializeMsg(pt uniproto.ProtocolType, pkt uint8, pn pulse.Number, msg string) []byte {
	packet := uniproto.NewSendingPacket(nil, nil)
	packet.Header.SetProtocolType(pt)
	packet.Header.SetPacketType(pkt)
	packet.Header.SetRelayRestricted(true)
	packet.PulseNumber = pn

	b, err := packet.SerializeToBytes(uint(len(msg)), func(_ nwapi.SerializationContext, _ *uniproto.Packet, w *iokit.LimitedWriter) error {
		_, err := w.Write([]byte(msg))
		return err
	})
	if err != nil {
		panic(throw.ErrorWithStack(err))
	}
	return b
}
