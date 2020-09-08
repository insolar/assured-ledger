// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/defaults"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insconveyor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/logwatermill"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	payload "github.com/insolar/assured-ledger/ledger-core/rms"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var (
	machineConfig = smachine.SlotMachineConfig{
		PollingPeriod:   500 * time.Millisecond,
		PollingTruncate: 1 * time.Millisecond,
		SlotPageSize:    1000,
		ScanCountLimit:  100000,
	}
)

func newPublisherForConveyor(factoryFn conveyor.PulseEventFactoryFunc) message.NoPublishHandlerFunc {
	ctx := context.Background()
	pulseConveyor := conveyor.NewPulseConveyor(ctx, conveyor.PulseConveyorConfig{
		ConveyorMachineConfig: machineConfig,
		SlotMachineConfig:     machineConfig,
		EventlessSleep:        0,
		MinCachePulseAge:      100,
		MaxPastPulseAge:       1000,
	}, factoryFn, nil)

	disp := insconveyor.NewConveyorDispatcher(ctx, pulseConveyor)

	return func(msg *message.Message) error {
		bm := beat.NewMessageExt(msg.UUID, msg.Payload, msg)
		bm.Metadata = msg.Metadata
		return disp.Process(bm)
	}
}

type watermillLogErrorHandler func(logger *logwatermill.WatermillLogAdapter, msg string, err error, fields watermill.LogFields) bool

func newWatermillLogAdapterWrapper(log log.Logger, errorHandler watermillLogErrorHandler) *watermillLogAdapterWrapper {
	return &watermillLogAdapterWrapper{
		WatermillLogAdapter: *logwatermill.NewWatermillLogAdapter(log),
		errorHandler:        errorHandler,
	}
}

type watermillLogAdapterWrapper struct {
	logwatermill.WatermillLogAdapter
	errorHandler watermillLogErrorHandler
}

func (w *watermillLogAdapterWrapper) Error(msg string, err error, fields watermill.LogFields) {
	if w.errorHandler != nil && w.errorHandler(&w.WatermillLogAdapter, msg, err, fields) {
		return
	}
	w.WatermillLogAdapter.Error(msg, err, fields)
}

func wait(timeout time.Duration, fn func() bool) bool {
	deltaTime := time.Millisecond
	startTime := time.Now()

	for {
		if fn() {
			return true
		}

		if time.Now().Sub(startTime) > timeout {
			return false
		}

		time.Sleep(deltaTime)
		deltaTime *= 2
	}
}

func TestWatermill_HandleErrorCorrect(t *testing.T) {
	defer commontestutils.LeakTester(t)

	const errorMsg = "handler error"
	watermillErrorHandler := func(logger *logwatermill.WatermillLogAdapter, msg string, err error, fields watermill.LogFields) bool {
		if err.Error() == errorMsg {
			logger.Info(msg+" | Error: "+err.Error(), fields)
			return true
		}
		return false
	}
	var (
		ctx        = context.Background()
		wmLogger   = newWatermillLogAdapterWrapper(inslogger.FromContext(ctx), watermillErrorHandler)
		subscriber = gochannel.NewGoChannel(gochannel.Config{}, wmLogger)

		cnt atomickit.Int
	)

	dispatchFn := func(context.Context, conveyor.InputEvent, conveyor.InputContext) (conveyor.InputSetup, error) {
		cnt.Add(1)
		return conveyor.InputSetup{}, throw.E(errorMsg)
	}

	wmStop := startWatermill(ctx, wmLogger, subscriber, newPublisherForConveyor(dispatchFn))
	defer wmStop()

	meta := payload.Meta{Pulse: pulse.Number(pulse.MinTimePulse + 1)}
	metaPl, _ := meta.Marshal()
	msg := message.NewMessage("1", metaPl)
	require.NoError(t, subscriber.Publish(defaults.TopicIncoming, msg))

	wait(time.Second, func() bool { return cnt.Load() >= 1 })
	require.Equal(t, 1, cnt.Load())
}

func TestWatermill_HandlePanicCorrect(t *testing.T) {
	defer commontestutils.LeakTester(t)

	const panicMsg = "handler panic"
	watermillErrorHandler := func(logger *logwatermill.WatermillLogAdapter, msg string, err error, fields watermill.LogFields) bool {
		if err.Error() == panicMsg {
			logger.Info(msg+" | Error: "+err.Error(), fields)
			return true
		}
		return false
	}
	var (
		ctx        = context.Background()
		wmLogger   = newWatermillLogAdapterWrapper(inslogger.FromContext(ctx), watermillErrorHandler)
		subscriber = gochannel.NewGoChannel(gochannel.Config{}, wmLogger)

		cnt atomickit.Int
	)

	dispatchFn := func(context.Context, conveyor.InputEvent, conveyor.InputContext) (conveyor.InputSetup, error) {
		cnt.Add(1)
		panic(throw.E(panicMsg))
	}

	wmStop := startWatermill(ctx, wmLogger, subscriber, newPublisherForConveyor(dispatchFn))
	defer wmStop()

	meta := payload.Meta{Pulse: pulse.Number(pulse.MinTimePulse + 1)}
	metaPl, _ := meta.Marshal()
	msg := message.NewMessage("1", metaPl)
	require.NoError(t, subscriber.Publish(defaults.TopicIncoming, msg))

	wait(time.Second, func() bool { return cnt.Load() >= 1 })
	require.Equal(t, 1, cnt.Load())
}

func startWatermill(
	ctx context.Context,
	logger watermill.LoggerAdapter,
	sub message.Subscriber,
	inHandler message.NoPublishHandlerFunc,
) func() {
	inRouter, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		panic(err)
	}
	inRouter.AddNoPublisherHandler(
		"IncomingHandler",
		defaults.TopicIncoming,
		sub,
		inHandler,
	)
	startRouter(ctx, inRouter)
	return func() {
		if inRouter.Close() != nil {
			inslogger.FromContext(ctx).Error("Error while closing router", err)
		}
	}
}

func startRouter(ctx context.Context, router *message.Router) {
	go func() {
		if err := router.Run(ctx); err != nil {
			inslogger.FromContext(ctx).Error("Error while running router", err)
		}
	}()
	<-router.Running()
}
