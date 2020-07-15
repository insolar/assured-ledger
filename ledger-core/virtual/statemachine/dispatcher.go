// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package statemachine

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/insolar/assured-ledger/ledger-core/appctl"
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

var _ appctl.Dispatcher = &conveyorDispatcher{}

func (c *conveyorDispatcher) PreparePulseChange(change appctl.PulseChange, sink appctl.NodeStateSink) {
	stateChan := sink.Occupy()

	if c.prevPulse.IsUnknown() {
		stateChan <- appctl.NodeState{}
		return
	}

	if err := c.conveyor.PreparePulseChange(stateChan); err != nil {
		panic(throw.WithStack(err))
	}
}

func (c *conveyorDispatcher) CancelPulseChange() {
	if c.prevPulse.IsUnknown() {
		return
	}
	if err := c.conveyor.CancelPulseChange(); err != nil {
		panic(throw.WithStack(err))
	}
}

func (c *conveyorDispatcher) CommitPulseChange(change appctl.PulseChange) {
	if err := c.conveyor.CommitPulseChange(change.Pulse, change.StartedAt); err != nil {
		panic(throw.WithStack(err))
	}
	c.prevPulse = change.Pulse.RightBoundData().PulseNumber
}

type DispatchedMessage struct {
	MessageMeta message.Metadata
	PayloadMeta payload.Meta
}

func (c *conveyorDispatcher) Process(msg *message.Message) error {
	msg.Ack()
	dm := DispatchedMessage{ MessageMeta: msg.Metadata }

	if err := rms.UnmarshalAs(msg.Payload, &dm.PayloadMeta, nil); err != nil {
		return throw.W(err, "failed to unmarshal payload.Meta")
	}
	return c.conveyor.AddInput(c.ctx, dm.PayloadMeta.Pulse, dm)
}

func NewConveyorDispatcher(ctx context.Context, conveyor *conveyor.PulseConveyor) appctl.Dispatcher {
	return &conveyorDispatcher{ ctx: ctx, conveyor: conveyor}
}
