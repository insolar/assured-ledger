package smachine

import (
	"fmt"
)

// Both prepareSubroutineInit and prepareSlotInit MUST be in-line
func (s *Slot) prepareSlotInit(creator *Slot, fn CreateFunc, sm StateMachine, defValues CreateDefaultValues) InitFunc {
	m := s.machine

	cc := constructionContext{creator: creator, s: s, injects: defValues.OverriddenDependencies, tracerID: defValues.TracerID}
	selfUpdate := s == creator
	if selfUpdate {
		cc.inherit = InheritResolvedDependencies
		cc.isTracing = creator.isTracing()
		if cc.tracerID == "" && s.stepLogger != nil {
			cc.tracerID = creator.stepLogger.GetTracerID()
		}
	}
	if defValues.InheritAllDependencies {
		cc.inherit = InheritAllDependencies
	}

	if fn != nil {
		sm = cc.executeCreate(fn)

		if sm == nil && selfUpdate {
			return nil
		}
		s._addTerminationCallback(cc.callbackLink, cc.callbackFn)
		if sm == nil {
			return nil
		}
	}
	s.setTracing(cc.isTracing)

	decl := sm.GetStateMachineDeclaration()
	if decl == nil {
		panic(fmt.Errorf("illegal state - declaration is missing: %v", sm))
	}
	s.declaration = decl

	// get injects sorted out
	var localInjects []interface{}
	{
		var creatorInheritable map[string]interface{}
		if creator != nil {
			creatorInheritable = creator.inheritable
		}
		s.inheritable, localInjects = m.prepareInjects(s.NewLink(), sm, cc.inherit, selfUpdate,
			creatorInheritable, cc.injects)
	}

	// Step Logger
	m.prepareStepLogger(s, sm, cc.tracerID)

	// get Init step
	initFn := decl.GetInitStateFor(sm)
	if initFn == nil {
		panic(fmt.Errorf("illegal state - initialization is missing: %v", sm))
	}

	initFn = preparePreInit(initFn, defValues.PreInitializationHandler, sm)

	// Setup Slot counters etc
	switch {
	case creator == nil:
		{
			scanCount, migrateCount := m.getScanAndMigrateCounts()
			s.migrationCount = migrateCount
			s.lastWorkScan = uint8(scanCount)
		}

		smInitFn := initFn
		initFn = func(ctx InitializationContext) StateUpdate {
			// when creator == nil slot addition is done asynchronously
			// so we have to make sure, that migrate and scan counts
			// are matched the time when init is executed, not when the slot was allocated

			_, migrateCount := s.machine.getScanAndMigrateCounts()
			if n := migrateCount - s.migrationCount; n > 0 {
				s.runShadowMigrate(n)
			}
			s.migrationCount = migrateCount
			return smInitFn(ctx)
		}
	case selfUpdate:
		// can't inherit SM-bound handler
		s.defMigrate = nil
		s.defErrorHandler = nil
		s.defFlags = 0
	default:
		s.migrationCount = creator.migrationCount
		s.lastWorkScan = creator.lastWorkScan
	}

	// shadow migrate for injected dependencies
	s.shadowMigrate = buildShadowMigrator(localInjects, s.declaration.GetShadowMigrateFor(sm))

	return initFn
}

func preparePreInit(initFn InitFunc, preInitFn PreInitHandlerFunc, sm StateMachine) InitFunc {
	if preInitFn == nil {
		return initFn
	}
	return func(ctx InitializationContext) StateUpdate {
		postError := preInitFn(ctx, sm)
		su := initFn(ctx)
		if postError != nil {
			return ctx.Error(postError)
		}
		return su
	}
}

// Both prepareSubroutineInit and prepareSlotInit MUST be in-line
func (s *Slot) prepareSubroutineInit(sm SubroutineStateMachine, tracerID TracerID) InitFunc { // nolint:interfacer
	m := s.machine
	prev := s.stateStack

	s.defFlags = prev.defFlags
	// get injects sorted out
	var localInjects []interface{}
	s.inheritable, localInjects = m.prepareInjects(s.NewLink(), sm, InheritResolvedDependencies,
		true, prev.inheritable, nil)

	// Step Logger
	m.prepareStepLogger(s, sm, tracerID)

	sc := subroutineStartContext{slotContext{s: s}, 0}
	initFn := sc.executeSubroutineStart(sm.GetSubroutineInitState)

	if initFn == nil {
		panic(fmt.Errorf("illegal state - initialization is missing: %v", sm))
	}

	var aliases *slotAliases
	if prev.stateStack != nil {
		aliases = prev.stateStack.copyAliases
	}
	prev.cleanupMode = sc.cleanupMode
	prev.copyAliases = s.storeSubroutineAliases(aliases, sc.cleanupMode)

	// shadow migrate for injected dependencies
	s.shadowMigrate = buildShadowMigrator(localInjects, s.declaration.GetShadowMigrateFor(sm))

	return initFn
}

func (s *Slot) prepareReplace(fn CreateFunc, sm StateMachine, defValues CreateDefaultValues) StateFunc {
	if initFn := s.prepareSlotInit(s, fn, sm, defValues); initFn != nil {
		s.slotFlags |= slotStepSuspendMigrate
		return initFn.defaultInit
	}
	panic("replacing SM didn't initialize")
}

var replaceInitDecl = StepDeclaration{stepDeclExt: stepDeclExt{Name: "<init_replace>"}}
var defaultInitDecl = StepDeclaration{stepDeclExt: stepDeclExt{Name: "<init>"}}

func (v InitFunc) defaultInit(ctx ExecutionContext) StateUpdate {
	ec := ctx.(*executionContext)
	if ec.s.shadowMigrate != nil {
		ec.s.shadowMigrate(ec.s.migrationCount, 0)
	}
	ic := initializationContext{ec.clone(updCtxInactive)}
	su := ic.executeInitialization(v)
	su.marker = ec.getMarker()
	return su
}
