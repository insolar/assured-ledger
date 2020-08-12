// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

type StepFlags uint16

const (
	// StepResetAllFlags indicates that Slot's default flags (set by SetDefaultFlags()) will be ignored, otherwise ORed.
	StepResetAllFlags StepFlags = 1 << iota

	// StepWeak marks a "weak" step - SlotMachine can be stopped automatically when all SMs are at "weak" steps.
	StepWeak

	// StepPriority makes SM's step to be executed before other steps in a cycle. Implies StepSyncPriority.
	StepPriority

	// StepSyncBoost makes SM's step to be handled as boosted for sync queues, but doesn't prioritise step in SlotMachine.
	StepSyncBoost

	// StepElevatedLog informs logger to provide alternative (e.g. higher levels) for output of this step.
	StepElevatedLog

	// stepSleepState is used to restore sleep status on RestoreStep() after AffectedStep
	stepSleepState
)

// Describes a step of a SM
type SlotStep struct {
	// Function to be called when the step is executed. MUST NOT be nil
	Transition StateFunc

	// Function to be called for migration of this step. Overrides SetDefaultMigration() when not nil.
	Migration MigrateFunc

	// Function to be called to handler errors of this step. Overrides SetDefaultErrorHandler() when not nil.
	Handler ErrorHandlerFunc

	// Step will be executed with the given flags. When StepResetAllFlags is specified, then SetDefaultFlags() is ignored, otherwise ORed.
	Flags StepFlags
}

func (s SlotStep) IsZero() bool {
	return s.Transition == nil && s.Flags == 0 && s.Migration == nil && s.Handler == nil
}

func (s SlotStep) ensureTransition() {
	if s.Transition == nil {
		panic("illegal value")
	}
}

func (s SlotStep) NoSleep() SlotStep {
	s.Flags &^= stepSleepState
	return s
}
