// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insconveyor

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type conveyorDispatcher struct {
	ctx       context.Context
	conveyor  *conveyor.PulseConveyor
	prevPulse pulse.Number
}

var _ beat.Dispatcher = &conveyorDispatcher{}

func (c *conveyorDispatcher) PrepareBeat(sink beat.Ack) {
	stateFn := sink.Acquire()

	if c.prevPulse.IsUnknown() {
		// Conveyor can't prepare without an initial pulse - there are no active SMs inside
		stateFn(beat.AckData{})
		return
	}

	if err := c.conveyor.PreparePulseChange(stateFn); err != nil {
		panic(throw.WithStack(err))
	}
}

func (c *conveyorDispatcher) CancelBeat() {
	if c.prevPulse.IsUnknown() {
		// Conveyor can't prepare without an initial pulse - there are no active SMs inside
		return
	}
	if err := c.conveyor.CancelPulseChange(); err != nil {
		panic(throw.WithStack(err))
	}
}

func (c *conveyorDispatcher) CommitBeat(change beat.Beat) {
	pulseRange := change.Range

	if pulseRange == nil {
		switch pn, ok := change.PulseNumber.TryPrev(change.PrevPulseDelta); {
		case ok && pn == c.prevPulse:
			pulseRange = change.AsRange()
		case c.prevPulse.IsUnknown():
			pulseRange = pulse.NewLeftGapRange(pulse.MinTimePulse, 0, change.Data)
		default:
			pulseRange = pulse.NewLeftGapRange(c.prevPulse, 0, change.Data)
		}
	} else if !c.prevPulse.IsUnknown() {
		prevPulse, ok := pulseRange.LeftBoundNumber().TryPrev(pulseRange.LeftPrevDelta())
		if !ok || prevPulse != c.prevPulse {
			panic(throw.IllegalState())
		}
	}

	if err := c.conveyor.CommitPulseChange(pulseRange, change.StartedAt, change.Online); err != nil {
		panic(throw.WithStack(err))
	}
	c.prevPulse = change.PulseNumber
}

type DispatchedMessage struct {
	MessageMeta message.Metadata
	PayloadMeta payload.Meta
}

func (c *conveyorDispatcher) Process(msg beat.Message) error {
	msg.Ack()
	dm := DispatchedMessage{ MessageMeta: msg.Metadata }

	if err := rms.UnmarshalAs(msg.Payload, &dm.PayloadMeta, nil); err != nil {
		return throw.W(err, "failed to unmarshal payload.Meta")
	}
	return c.conveyor.AddInput(c.ctx, dm.PayloadMeta.Pulse, dm)
}

func NewConveyorDispatcher(ctx context.Context, conveyor *conveyor.PulseConveyor) beat.Dispatcher {
	return &conveyorDispatcher{ ctx: ctx, conveyor: conveyor}
}
