package smachine

import (
	"fmt"
	"math"
)

const UnknownSlotID SlotID = 0

type SlotID uint32

func (id SlotID) IsUnknown() bool {
	return id == UnknownSlotID
}

func NoLink() SlotLink {
	return SlotLink{}
}

func NoStepLink() StepLink {
	return StepLink{}
}

var deadSlot = Slot{}

func DeadSlotLink() SlotLink {
	return SlotLink{id: 1, s: &deadSlot}
}

func DeadStepLink() StepLink {
	return StepLink{SlotLink: DeadSlotLink(), step: numberOfReservedSteps}
}

// A lazy-like link to a slot, that can detect if a slot is dead */
type SlotLink struct {
	id SlotID
	s  *Slot
}

func (p SlotLink) String() string {
	if p.s == nil {
		if p.id == 0 {
			return "<nil>"
		}
		return fmt.Sprintf("noslot-%d", p.id)
	}
	return fmt.Sprintf("slot-%d", p.id)
}

func (p SlotLink) MachineID() string {
	return p.getMachine().GetMachineID()
}

func (p SlotLink) SlotID() SlotID {
	return p.id
}

func (p SlotLink) IsZero() bool {
	return p.s == nil
}

// Returns true when the slot is valid/alive at the given moment
func (p SlotLink) IsValid() bool {
	if p.s == nil {
		return false
	}
	id, _, _ := p.s._getState()
	return p.id == id
}

// Returns a link to a step the slot is at.
// For an invalid valid slot returns an unbound StepLink (zero step) to the the same slot.
func (p SlotLink) GetStepLink() (stepLink StepLink, isValid bool) {
	if p.s == nil {
		return StepLink{}, false
	}
	id, step, _ := p.s._getState()
	if p.id == id {
		return StepLink{p, step}, true
	}
	return StepLink{p, 0}, false
}

func (p SlotLink) GetAnyStepLink() StepLink {
	if p.s == nil {
		return StepLink{}
	}
	return StepLink{p, 0}
}

func (p SlotLink) isValidAndBusy() bool {
	if p.s == nil {
		return false
	}
	id, _, isBusy := p.s._getState()
	return p.id == id && isBusy
}

func (p SlotLink) getIsValidAndBusy() (isValid, isBusy bool) {
	if p.s == nil {
		return false, false
	}
	id, _, isBusy := p.s._getState()
	return p.id == id, isBusy
}

func (p SlotLink) tryStartWorking() (s *Slot, isStarted bool, prevStepNo uint32) {
	if p.s != nil {
		if _, isStarted, prevStepNo = p.s._tryStartWithID(p.id, 1); isStarted {
			return p.s, true, prevStepNo
		}
	}
	return nil, false, 0
}

func (p SlotLink) isMachine(m *SlotMachine) bool {
	return m != nil && m == p.getMachine()
}

func (p SlotLink) getActiveMachine() *SlotMachine {
	if m := p.getMachine(); m != nil && m.IsActive() {
		return m
	}
	return nil
}

func (p SlotLink) getMachine() *SlotMachine {
	if p.s != nil {
		return p.s.getMachine()
	}
	return nil
}

const numberOfReservedSteps = 2

type StepLink struct {
	SlotLink
	step uint32
}

func (p StepLink) StepNo() uint32 {
	return p.step
}

// Makes the step link unbound - IsAtStep()/IsNearStep() will return true for any step of a valid slot
func (p StepLink) AnyStep() StepLink {
	if p.step != 0 {
		p.step = 0
	}
	return p
}

func (p StepLink) String() string {
	if p.step == 0 {
		return p.SlotLink.String()
	}
	return fmt.Sprintf("%s-step-%d", p.SlotLink.String(), p.id)
}

// Returns true when the slot is valid and either the slot stays at the same step or this StepLink is unbound
func (p StepLink) IsAtStep() bool {
	if p.s == nil {
		return false
	}
	id, step, _ := p.s._getState()
	return p.id == id && (p.step == 0 || p.step == step)
}

// Same as IsAtStep(), but return true when the slot's step is [link.step .. link.step+forwardDelta], safe for step wrapping
func (p StepLink) IsNearStep(forwardDelta uint32) bool {
	if p.s == nil {
		return false
	}
	switch id, step, _ := p.s._getState(); {
	case p.id != id:
		return false
	case p.step == 0:
		return true
	case forwardDelta == 0:
		return p.step == step
	default:
		switch lastStep := p.step + forwardDelta; {
		case p.step <= lastStep: // no overflow
			return p.step <= step && step <= lastStep
		case forwardDelta >= math.MaxUint32-numberOfReservedSteps:
			return true
		default: // overflow
			lastStep += numberOfReservedSteps
			return p.step <= step || step <= lastStep
		}
	}
}

func (p StepLink) isValidAndAtExactStep() (valid, atExactStep bool) { // nolint:unparam
	if p.s == nil {
		return false, false
	}
	id, step, _ := p.s._getState()
	return p.id == id, p.step == step
}

// func (p StepLink) getIsValidBusyAndAtStep() (isValid, isBusy, atExactStep bool) {
// 	if p.s == nil {
// 		return false, false, false
// 	}
// 	id, step, isBusy := p.s._getState()
// 	return p.id == id, isBusy, p.step == 0 || p.step == step
// }
