// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package statemachine

import (
	"context"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/dispatcher"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
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

func newDispatcherWithConveyor(factoryFn conveyor.PulseEventFactoryFunc) dispatcher.Dispatcher {
	ctx := context.Background()
	pulseConveyor := conveyor.NewPulseConveyor(ctx, conveyor.PulseConveyorConfig{
		ConveyorMachineConfig: machineConfig,
		SlotMachineConfig:     machineConfig,
		EventlessSleep:        0,
		MinCachePulseAge:      100,
		MaxPastPulseAge:       1000,
	}, factoryFn, nil)
	return NewConveyorDispatcher(pulseConveyor)
}

func TestConveyorDispatcher_ErrorUnmarshalHandling(t *testing.T) {
	dispatcher := newDispatcherWithConveyor(nil)
	require.NotPanics(t, func() {
		require.NoError(t, dispatcher.Process(&message.Message{}))
	})
}

func TestConveyorDispatcher_WrongMetaTypeHandling(t *testing.T) {
	dispatcher := newDispatcherWithConveyor(nil)
	req := payload.VCallRequest{}
	pl, _ := req.Marshal()
	require.NotPanics(t, func() {
		require.NoError(t, dispatcher.Process(&message.Message{Payload: pl}))
	})
}

func TestConveyorDispatcher_PanicInAddInputHandling(t *testing.T) {
	dispatcher := newDispatcherWithConveyor(
		func(_ pulse.Number, _ pulse.Range, _ conveyor.InputEvent) (pulse.Number, smachine.CreateFunc, error) {
			panic(throw.E("handler panic"))
		})
	meta := payload.Meta{Pulse: pulse.Number(pulse.MinTimePulse + 1)}
	metaPl, _ := meta.Marshal()
	require.NotPanics(t, func() {
		require.NoError(t, dispatcher.Process(&message.Message{Payload: metaPl}))
	})
}

func TestConveyorDispatcher_ErrorInAddInputHandling(t *testing.T) {
	dispatcher := newDispatcherWithConveyor(
		func(_ pulse.Number, _ pulse.Range, _ conveyor.InputEvent) (pulse.Number, smachine.CreateFunc, error) {
			return 0, nil, throw.E("handler error")

		})
	meta := payload.Meta{Pulse: pulse.Number(pulse.MinTimePulse + 1)}
	metaPl, _ := meta.Marshal()
	metadata := message.Metadata{}
	metadata.Set("error_type", "error")
	require.NotPanics(t, func() {
		require.NoError(t, dispatcher.Process(&message.Message{Payload: metaPl, Metadata: metadata}))
	})
}
