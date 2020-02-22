/*
 * Copyright 2020 Insolar Network Ltd.
 * All rights reserved.
 * This material is licensed under the Insolar License version 1.0,
 * available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package smachine

import (
	"fmt"
)

// Both prepareSubroutineInit and prepareSlotInit MUST be in-line
func (s *Slot) prepareSlotInit(creator *Slot, fn CreateFunc, sm StateMachine, defValues CreateDefaultValues) InitFunc {
	m := s.machine

	cc := constructionContext{s: s, injects: defValues.OverriddenDependencies, tracerId: defValues.TracerId}
	selfUpdate := s == creator
	if selfUpdate {
		cc.inherit = InheritResolvedDependencies
		cc.isTracing = creator.isTracing()
		if cc.tracerId == "" && s.stepLogger != nil {
			cc.tracerId = creator.stepLogger.GetTracerId()
		}
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
	m.prepareStepLogger(s, sm, cc.tracerId)

	// get Init step
	initFn := decl.GetInitStateFor(sm)
	if initFn == nil {
		panic(fmt.Errorf("illegal state - initialization is missing: %v", sm))
	}

	// Setup Slot counters
	switch {
	case creator == nil:
		scanCount, migrateCount := m.getScanAndMigrateCounts()
		s.migrationCount = migrateCount
		s.lastWorkScan = uint8(scanCount)
	case selfUpdate:
		//
	default:
		s.migrationCount = creator.migrationCount
		s.lastWorkScan = creator.lastWorkScan
	}

	// shadow migrate for injected dependencies
	s.shadowMigrate = buildShadowMigrator(localInjects, s.declaration.GetShadowMigrateFor(sm))

	return initFn
}

// Both prepareSubroutineInit and prepareSlotInit MUST be in-line
func (s *Slot) prepareSubroutineInit(sm SubroutineStateMachine, tracerId TracerId) InitFunc {
	m := s.machine
	// get injects sorted out
	var localInjects []interface{}
	s.inheritable, localInjects = m.prepareInjects(s.NewLink(), sm, InheritResolvedDependencies,
		true, s.inheritable, nil)

	initFn := sm.GetSubroutineInitState()
	if initFn == nil {
		panic(fmt.Errorf("illegal state - initialization is missing: %v", sm))
	}

	// Step Logger
	m.prepareStepLogger(s, sm, tracerId)

	// shadow migrate for injected dependencies
	s.shadowMigrate = buildShadowMigrator(localInjects, s.declaration.GetShadowMigrateFor(sm))

	return initFn
}

var replaceInitDecl = StepDeclaration{stepDeclExt: stepDeclExt{Name: "<replace init>"}}
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
