package smachine

import (
	"context"
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/reflectkit"
)

type StateMachine interface {
	// GetStateMachineDeclaration returns a meta-type / declaration of a SM.
	// Must be non-nil.
	GetStateMachineDeclaration() StateMachineDeclaration
}

type StateMachineDeclaration interface {
	StateMachineHelper
	// GetInitStateFor an initialization function for the given SM.
	// Is called once per SM after InjectDependencies().
	GetInitStateFor(StateMachine) InitFunc
}

type StateMachineHelper interface {
	// InjectDependencies runs initialization code that SM doesn't need to know.
	// Is called once per SM right after GetStateMachineDeclaration().
	// Dependencies injected through DependencyInjector and implementing ShadowMigrator will be invoked during migration.
	InjectDependencies(StateMachine, SlotLink, injector.DependencyInjector)

	// GetStepLogger provides per-SM logger. Zero implementation must return (nil, false).
	// Is called once per SM after InjectDependencies().
	// When result is (_, false) then StepLoggerFactory will be used.
	// When result is (nil, true) then any logging will be disabled.
	GetStepLogger(context.Context, StateMachine, TracerID, StepLoggerFactoryFunc) (StepLogger, bool)

	// GetShadowMigrateFor returns a shadow migration handler for the given stateMachine, that will be invoked on every migration. SM has no control over it.
	// Is called once per SM after InjectDependencies().
	// See ShadowMigrator
	GetShadowMigrateFor(StateMachine) ShadowMigrateFunc

	// GetStepDeclaration returns a StepDeclaration for the given step. Return nil when implementation is not available.
	GetStepDeclaration(StateFunc) *StepDeclaration

	// IsConsecutive is only invoked when GetStepDeclaration() is not available for the current step.
	// WARNING! DO NOT EVER return "true" here without CLEAR understanding of internal mechanics.
	// Returning "true" blindly will LIKELY lead to infinite loops.
	IsConsecutive(cur, next StateFunc) bool
}

type SubroutineStateMachine interface {
	StateMachine
	GetSubroutineInitState(SubroutineStartContext) InitFunc
}

type stepDeclExt struct {
	SeqID int
	Name  string
}

type StepDeclaration struct {
	SlotStep
	stepDeclExt
}

func (v StepDeclaration) GetStepName() string {
	switch {
	case len(v.Name) > 0:
		if v.SeqID != 0 {
			return fmt.Sprintf("%s[%d]", v.Name, v.SeqID)
		}
		return fmt.Sprint(v.Name)

	case v.SeqID != 0:
		return fmt.Sprintf("#[%d]", v.SeqID)

	case v.Transition == nil:
		return "<nil>"

	default:
		return fmt.Sprintf("%p", v.Transition)
	}
}

func (v StepDeclaration) IsNameless() bool {
	return v.SeqID == 0 && len(v.Name) == 0
}

// See ShadowMigrator
type ShadowMigrateFunc func(migrationCount, migrationDelta uint32)

// Provides assistance to injected and other objects handle migration events.
type ShadowMigrator interface {
	// Called on migration of a related slot BEFORE every call to a normal migration handler with migrationDelta=1.
	// When there is no migration handler is present or SkipMultipleMigrations() was used, then an additional call is made
	// with migrationDelta > 0 to indicate how many migration steps were skipped.
	ShadowMigrate(migrationCount, migrationDelta uint32)
}

// A template to include into SM to avoid hassle of creation of any methods but GetInitStateFor()
type StateMachineDeclTemplate struct {
}

func (s *StateMachineDeclTemplate) GetStepDeclaration(StateFunc) *StepDeclaration {
	return nil
}

func (s *StateMachineDeclTemplate) IsConsecutive(cur, next StateFunc) bool {
	return reflectkit.CodeOf(cur) < reflectkit.CodeOf(next)
}

func (s *StateMachineDeclTemplate) GetShadowMigrateFor(StateMachine) ShadowMigrateFunc {
	return nil
}

func (s *StateMachineDeclTemplate) InjectDependencies(StateMachine, SlotLink, injector.DependencyInjector) {
}

func (s *StateMachineDeclTemplate) GetStepLogger(context.Context, StateMachine, TracerID, StepLoggerFactoryFunc) (StepLogger, bool) {
	return nil, false
}

type TerminationHandlerFunc func(TerminationData)

// (FixedSlotWorker) arg can be nil
type internalTerminationHandlerFunc func(TerminationData, FixedSlotWorker)

type TerminationData struct {
	Slot    StepLink
	Parent  SlotLink
	Context context.Context
	Result  interface{}
	Error   error
}

type PreInitHandlerFunc = func (InitializationContext, StateMachine) (postInitError error)

type CreateDefaultValues struct {
	Context                context.Context
	Parent                 SlotLink
	OverriddenDependencies map[string]interface{}
	InheritAllDependencies bool

	PreInitializationHandler PreInitHandlerFunc

	// TerminationHandler provides a special termination handler that will be invoked AFTER termination of SM.
	// This handler is invoked with data from GetTerminationResult() and error (if any).
	// This handler is not directly accessible to SM.
	// WARNING! This handler is UNSAFE to access any SM.
	TerminationHandler TerminationHandlerFunc
	TerminationResult  interface{}
	TracerID           TracerID
}

func (p *CreateDefaultValues) PutOverride(id string, v interface{}) {
	if id == "" {
		panic("illegal value")
	}
	if p.OverriddenDependencies == nil {
		p.OverriddenDependencies = map[string]interface{}{id: v}
	} else {
		p.OverriddenDependencies[id] = v
	}
}
