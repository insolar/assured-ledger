package adapters

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/transport"
)

func ConsensusContext(ctx context.Context) context.Context {
	return inslogger.UpdateLogger(ctx, func(logger log.Logger) (log.Logger, error) {
		return logger.Copy().WithFields(map[string]interface{}{
			"component":  "consensus",
			"LowLatency": true,
		}).WithMetrics(logcommon.LogMetricsWriteDelayField).BuildLowLatency()
	})
}

func PacketEarlyLogger(ctx context.Context, senderAddr string) (context.Context, log.Logger) {
	ctx = ConsensusContext(ctx)

	ctx, logger := inslogger.WithFields(ctx, map[string]interface{}{
		"sender_address": senderAddr,
	})

	return ctx, logger
}

func PacketLateLogger(ctx context.Context, parser transport.PacketParser) (context.Context, log.Logger) {
	ctx, logger := inslogger.WithFields(ctx, map[string]interface{}{
		"sender_id":    parser.GetSourceID(),
		"packet_type":  parser.GetPacketType().String(),
		"packet_pulse": parser.GetPulseNumber(),
	})

	return ctx, logger
}

func ReportContext(report api.UpstreamReport) context.Context {
	return network.NewPulseContext(context.Background(), uint32(report.PulseNumber))
}
