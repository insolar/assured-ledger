// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package statemachine

import (
	"context"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/bus/meta"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/flow/dispatcher"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/adapters"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
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

type logBeginPulseMessage struct {
	*log.Msg `fmt:"BeginPulse"`

	PreviousPulse insolar.PulseNumber
	NextPulse     insolar.PulseNumber `opt:""`
}

type logClosePulseMessage struct {
	*log.Msg `fmt:"ClosePulse"`

	PreviousPulse insolar.PulseNumber
}

func (c *conveyorDispatcher) BeginPulse(ctx context.Context, pulseObject insolar.Pulse) {
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
		panic(throw.Impossible())
	}

	inslogger.FromContext(ctx).Errorm(logBeginPulseMessage{
		PreviousPulse: c.previousPulse,
		NextPulse:     pulseData.PulseNumber,
	})

	// TODO pass proper pulse start time from consensus
	if err := c.conveyor.CommitPulseChange(pulseRange, time.Now()); err != nil {
		panic(err)
	}
}

func (c *conveyorDispatcher) ClosePulse(ctx context.Context, pulseObject insolar.Pulse) {
	inslogger.FromContext(ctx).Errorm(logClosePulseMessage{
		PreviousPulse: c.previousPulse,
	})

	c.previousPulse = pulseObject.PulseNumber

	switch c.state {
	case InitializationDone:
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
		panic(throw.Impossible())
	}
}

type DispatcherMessage struct {
	MessageMeta message.Metadata
	PayloadMeta *payload.Meta
}

type errUnknownPayload struct {
	ExpectedType string
	GotType      interface{} `fmt:"%T"`
}

func (c *conveyorDispatcher) Process(msg *message.Message) error {
	pl, err := payload.Unmarshal(msg.Payload)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal payload.Meta")
	}
	plMeta, ok := pl.(*payload.Meta)
	if !ok {
		return throw.E("unexpected type", errUnknownPayload{ExpectedType: "payload.Meta", GotType: pl})
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
