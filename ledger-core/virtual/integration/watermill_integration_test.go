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

	"github.com/insolar/assured-ledger/ledger-core/appctl"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/defaults"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/logwatermill"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/statemachine"
)

var (
	machineConfig = smachine.SlotMachineConfig{
		PollingPeriod:   500 * time.Millisecond,
		PollingTruncate: 1 * time.Millisecond,
		SlotPageSize:    1000,
		ScanCountLimit:  100000,
	}
)

func newDispatcherWithConveyor(factoryFn conveyor.PulseEventFactoryFunc) appctl.Dispatcher {
	ctx := context.Background()
	pulseConveyor := conveyor.NewPulseConveyor(ctx, conveyor.PulseConveyorConfig{
		ConveyorMachineConfig: machineConfig,
		SlotMachineConfig:     machineConfig,
		EventlessSleep:        0,
		MinCachePulseAge:      100,
		MaxPastPulseAge:       1000,
	}, factoryFn, nil)
	return statemachine.NewConveyorDispatcher(ctx, pulseConveyor)
}

type WatermillLogErrorHandler func(logger *logwatermill.WatermillLogAdapter, msg string, err error, fields watermill.LogFields) bool

func NewWatermillLogAdapterWrapper(log log.Logger, errorHandler WatermillLogErrorHandler) *WatermillLogAdapterWrapper {
	return &WatermillLogAdapterWrapper{
		WatermillLogAdapter: *logwatermill.NewWatermillLogAdapter(log),
		errorHandler:        errorHandler,
	}
}

type WatermillLogAdapterWrapper struct {
	logwatermill.WatermillLogAdapter
	errorHandler WatermillLogErrorHandler
}

func (w *WatermillLogAdapterWrapper) Error(msg string, err error, fields watermill.LogFields) {
	if w.errorHandler != nil && w.errorHandler(&w.WatermillLogAdapter, msg, err, fields) {
		return
	}
	w.WatermillLogAdapter.Error(msg, err, fields)
}

func TestWatermill_HandleErrorCorrect(t *testing.T) {
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
		wmLogger   = NewWatermillLogAdapterWrapper(inslogger.FromContext(ctx), watermillErrorHandler)
		subscriber = gochannel.NewGoChannel(gochannel.Config{}, wmLogger)
	)
	cnt := 0
	conveyorDispatcher := newDispatcherWithConveyor(
		func(context.Context, conveyor.InputEvent, conveyor.InputContext) (conveyor.InputSetup, error) {
			cnt++
			return conveyor.InputSetup{}, throw.E(errorMsg)
		})
	wmStop := startWatermill(ctx, wmLogger, subscriber, conveyorDispatcher.Process)
	defer wmStop()
	meta := payload.Meta{Pulse: pulse.Number(pulse.MinTimePulse + 1)}
	metaPl, _ := meta.Marshal()
	msg := message.NewMessage("1", metaPl)
	require.NoError(t, subscriber.Publish(defaults.TopicIncoming, msg))
	time.Sleep(1 * time.Second)
	require.Equal(t, 1, cnt)
}

func TestWatermill_HandlePanicCorrect(t *testing.T) {
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
		wmLogger   = NewWatermillLogAdapterWrapper(inslogger.FromContext(ctx), watermillErrorHandler)
		subscriber = gochannel.NewGoChannel(gochannel.Config{}, wmLogger)
	)
	cnt := 0
	conveyorDispatcher := newDispatcherWithConveyor(
		func(context.Context, conveyor.InputEvent, conveyor.InputContext) (conveyor.InputSetup, error) {
			cnt++
			panic(throw.E(panicMsg))
		})

	wmStop := startWatermill(ctx, wmLogger, subscriber, conveyorDispatcher.Process)
	defer wmStop()
	meta := payload.Meta{Pulse: pulse.Number(pulse.MinTimePulse + 1)}
	metaPl, _ := meta.Marshal()
	msg := message.NewMessage("1", metaPl)
	require.NoError(t, subscriber.Publish(defaults.TopicIncoming, msg))
	time.Sleep(1 * time.Second)
	require.Equal(t, 1, cnt)
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
