// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package predicate

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/testutils/debuglogger"
)

func NewSlotLinkFilter(slotLink smachine.SlotLink) SlotLinkFilter {
	return SlotLinkFilter{slotLink: slotLink}
}

type SlotLinkFilter struct {
	slotLink smachine.SlotLink
}

func (s SlotLinkFilter) IsFromSlot(event debuglogger.UpdateEvent) bool {
	return s.slotLink.SlotID() == event.Data.StepNo.SlotID()
}

func (s SlotLinkFilter) AfterStop() Func {
	return func(event debuglogger.UpdateEvent) bool {
		return !s.slotLink.IsValid()
	}
}
