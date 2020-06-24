// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datawriter

import (
<<<<<<< HEAD
<<<<<<< HEAD
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
=======
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
>>>>>>> Ledger SMs
=======
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
>>>>>>> Ledger SMs
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type StreamDropKey pulse.Number

type StreamDropCataloger interface {
<<<<<<< HEAD
<<<<<<< HEAD
	GetOrCreate(ctx smachine.ExecutionContext, pn pulse.Number) *StreamSharedData
=======
	GetOrCreate(ctx smachine.ExecutionContext, pn pulse.Number) StreamDropDataLink
>>>>>>> Ledger SMs
=======
	GetOrCreate(ctx smachine.ExecutionContext, pn pulse.Number) StreamDropDataLink
>>>>>>> Ledger SMs
}

var _ StreamDropCataloger = &StreamDropCatalog{}
type StreamDropCatalog struct {}

<<<<<<< HEAD
<<<<<<< HEAD
func (*StreamDropCatalog) GetOrCreate(ctx smachine.ExecutionContext, pn pulse.Number) *StreamSharedData {
	sdl := ctx.GetPublishedLink(StreamDropKey(pn))
	if !sdl.IsZero() {
		return sdl.TryDirectAccess().(*StreamSharedData)
=======
=======
>>>>>>> Ledger SMs
func (*StreamDropCatalog) GetOrCreate(ctx smachine.ExecutionContext, pn pulse.Number) StreamDropDataLink {
	switch sdl := ctx.GetPublishedLink(StreamDropKey(pn)); {
	case sdl.IsZero():
		break
	case sdl.IsAssignableTo(&StreamDropDataLink{}):
		return StreamDropDataLink{sdl}
	default:
		return StreamDropDataLink{}
<<<<<<< HEAD
>>>>>>> Ledger SMs
=======
>>>>>>> Ledger SMs
	}

	ctx.InitChild(StreamDropCreate())

<<<<<<< HEAD
<<<<<<< HEAD
	sdl = ctx.GetPublishedLink(StreamDropKey(pn))
	if sdl.IsZero() {
		panic(throw.IllegalState())
	}
	return sdl.TryDirectAccess().(*StreamSharedData)
=======
=======
>>>>>>> Ledger SMs
	switch sdl := ctx.GetPublishedLink(StreamDropKey(pn)); {
	case sdl.IsZero():
		panic(throw.IllegalState())
	case sdl.IsAssignableTo(&StreamDropDataLink{}):
		return StreamDropDataLink{sdl}
	default:
		panic(throw.IllegalState())
	}
<<<<<<< HEAD
>>>>>>> Ledger SMs
=======
>>>>>>> Ledger SMs
}

func StreamDropCreate() smachine.CreateFunc {
	return func(smachine.ConstructionContext) smachine.StateMachine {
		return &SMStreamDropBuilder{}
	}
}

<<<<<<< HEAD
<<<<<<< HEAD
func RegisterStreamDrop(ctx smachine.SharedStateContext, pr pulse.Range) *StreamSharedData {
	if pr == nil {
		panic(throw.IllegalState())
	}

	sd := &StreamSharedData{
		ready: smsync.NewConditionalBool(false, fmt.Sprintf("StreamDrop{%d}.ready", ctx.SlotLink().SlotID())),
		pr: pr,
	}

	sdl := ctx.Share(sd, smachine.ShareDataUnbound)
	if !ctx.Publish(StreamDropKey(pr.RightBoundData().PulseNumber), sdl) {
		ctx.Unshare(sdl)
		return nil
	}
	return sd
}

=======
=======
>>>>>>> Ledger SMs
type StreamDropDataLink struct {
	smachine.SharedDataLink
}

func (v StreamDropDataLink) GetSharedData(ctx smachine.ExecutionContext) (sd *StreamSharedData) {
	if !v.SharedDataLink.IsUnbound() {
		panic(throw.IllegalState())
	}

	if v.SharedDataLink.PrepareAccess(func(i interface{}) (wakeup bool) {
		sd = i.(*StreamSharedData)
		return false
	}).TryUse(ctx).GetDecision() != smachine.Passed {
		panic(throw.Impossible())
	}

	return
}
<<<<<<< HEAD
>>>>>>> Ledger SMs
=======
>>>>>>> Ledger SMs
