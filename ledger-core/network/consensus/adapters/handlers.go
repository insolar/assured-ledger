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
