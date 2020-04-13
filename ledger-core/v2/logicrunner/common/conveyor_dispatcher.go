// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package common

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/bus/meta"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/flow/dispatcher"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/global"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/adapters"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
)

type dispatcherInitializationState int8

const (
	InitializationStarted dispatcherInitializationState = iota
	FirstPulseClosed
	InitializationDone // SecondPulseOpened
)

type conveyorDispatcher struct {
	conveyor      *conveyor.PulseConveyor
	state         dispatcherInitializationState
	previousPulse insolar.PulseNumber
}

var _ dispatcher.Dispatcher = &conveyorDispatcher{}

func (c *conveyorDispatcher) BeginPulse(ctx context.Context, pulseObject insolar.Pulse) {
	logger := inslogger.FromContext(ctx)
	var (
		pulseData  = adapters.NewPulseData(pulseObject)
		pulseRange pulse.Range
	)
	switch c.state {
	case InitializationDone:
		pulseRange = pulseData.AsRange()
	case FirstPulseClosed:
		pulseRange = pulse.NewLeftGapRange(c.previousPulse, 0, pulseData)
		c.state = InitializationDone
	case InitializationStarted:
		fallthrough
	default:
		panic("unreachable")
	}

	logger.Errorf("BeginPulse -> [%d, %d]", c.previousPulse, pulseData.PulseNumber)
	if err := c.conveyor.CommitPulseChange(pulseRange); err != nil {
		panic(err)
	}
}

func (c *conveyorDispatcher) ClosePulse(ctx context.Context, pulseObject insolar.Pulse) {
	global.Errorf("ClosePulse -> [%d]", pulseObject.PulseNumber)
	c.previousPulse = pulseObject.PulseNumber

	switch c.state {
	case InitializationDone:
		// channel := make(conveyor.PreparePulseChangeChannel, 1)
		channel := conveyor.PreparePulseChangeChannel(nil)
		if err := c.conveyor.PreparePulseChange(channel); err != nil {
			panic(err)
		}
	case InitializationStarted:
		c.state = FirstPulseClosed
		return
	case FirstPulseClosed:
		fallthrough
	default:
		panic("unreachable")
	}
}

type DispatcherMessage struct {
	MessageMeta message.Metadata
	PayloadMeta *payload.Meta
}

func (c *conveyorDispatcher) Process(msg *message.Message) error {
	pl, err := payload.Unmarshal(msg.Payload)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal payload.Meta")
	}
	plMeta, ok := pl.(*payload.Meta)
	if !ok {
		return errors.Errorf("unexpected type: %T (expected payload.Meta)", pl)
	}

	ctx, _ := inslogger.WithTraceField(context.Background(), msg.Metadata.Get(meta.TraceID))
	return c.conveyor.AddInput(ctx, plMeta.Pulse, &DispatcherMessage{
		MessageMeta: msg.Metadata,
		PayloadMeta: plMeta,
	})
}

func NewConveyorDispatcher(conveyor *conveyor.PulseConveyor) dispatcher.Dispatcher {
	return &conveyorDispatcher{conveyor: conveyor}
}

type TestApiCall struct {
	Payload payload.VCallRequest
	Response chan payload.VCallResult
}
