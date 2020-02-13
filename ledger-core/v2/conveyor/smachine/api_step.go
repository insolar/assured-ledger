// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

type StepFlags uint16

const (
	// Indicates that Slot's default flags (set by SetDefaultFlags()) will be ignored, otherwise ORed.
	StepResetAllFlags StepFlags = 1 << iota

	// When SM is at a step that StepWeak flag, then SM is considered as "weak".
	// SlotMachine will delete all "weak" SMs when there are no "non-weak" or working SMs left.
	StepWeak

	// A step with StepPriority flag will be executed before other steps in a cycle.
	StepPriority

	// A marker for logger to log this step without tracing
	StepElevatedLog

	//StepIgnoreAsyncWakeup
	//StepForceAsyncWakeup
)

// Describes a step of a SM
type SlotStep struct {
	// Function to be called when the step is executed. MUST NOT be nil
	Transition StateFunc

	// Function to be called for migration of this step. Overrides SetDefaultMigration() when not nil.
	Migration MigrateFunc

	// Step will be executed with the given flags. When StepResetAllFlags is specified, then SetDefaultFlags() is ignored, otherwise ORed.
	Flags StepFlags

	// Function to be called to handler errors of this step. Overrides SetDefaultErrorHandler() when not nil.
	Handler ErrorHandlerFunc
}

func (s *SlotStep) IsZero() bool {
	return s.Transition == nil && s.Flags == 0 && s.Migration == nil && s.Handler == nil
}

func (s *SlotStep) ensureTransition() {
	if s.Transition == nil {
		panic("illegal value")
	}
}
