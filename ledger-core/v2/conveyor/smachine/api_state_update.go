// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

type ContextMarker = uintptr

// Represents an update to be applied to SM.
// Content of this object is used internally. DO NOT interfere.
type StateUpdate struct {
	marker  ContextMarker
	link    *Slot
	param0  uint32
	param1  interface{}
	step    SlotStep
	updKind uint8
}

func (u StateUpdate) IsZero() bool {
	return u.marker == 0 && u.updKind == 0
}

// IsEmpty returns true when StateUpdate is zero or has "no-op" action
func (u StateUpdate) IsEmpty() bool {
	return u.updKind == 0 || u.marker == 0
}

func (u StateUpdate) getLink() SlotLink {
	if u.link == nil {
		return NoLink()
	}
	return SlotLink{SlotID(u.param0), u.link}
}

func (u StateUpdate) ensureMarker(marker ContextMarker) StateUpdate {
	if u.marker != marker {
		panic("illegal state")
	}
	return u
}
