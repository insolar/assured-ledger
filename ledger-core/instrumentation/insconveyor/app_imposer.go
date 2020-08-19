// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insconveyor

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/managed"
	"github.com/insolar/assured-ledger/ledger-core/testutils/journal"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
)

// ImposerFunc enables imposing of test-specific overrides on an app compartment.
type ImposerFunc = func(*ImposedParams)

// ImposedParams is a set of overrides available to ImposerFunc provided with instestconveyor.ServerTemplate.SetImposer()
type ImposedParams struct {
	// InitContext provides a context given into Init() of app component (not compartment).
	// This context is cancelled on stop.
	InitContext          context.Context
	// AppInject contains dependencies provided from Server to App component
	AppInject injector.DependencyInjector
	// CompartmentSetup is a configuration of conveyor-based app compartment.
	CompartmentSetup     AppCompartmentSetup
	// ComponentInterceptFn for use by ServerTemplate.InitTemplate.
	ComponentInterceptFn ComponentInterceptFunc
	// ConveyorCycleFn enables access to conveyor's worker state. See instestconveyor.CycleController
	ConveyorCycleFn      conveyor.PulseConveyorCycleFunc
	// EventJournal will be connected to logger of conveyor when is not nil.
	EventJournal         *journal.Journal
}

// ComponentInterceptFunc allows interception / replacement of conveyor-managed components of an app compartment.
type ComponentInterceptFunc = func(managed.Component) managed.Component
