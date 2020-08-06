// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insconveyor

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/managed"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
)

type ImposedParams struct {
	CompartmentSetup  AppCompartmentSetup
	ConveyorRegistry  injector.DependencyRegistry
	RegisterComponent managed.RegisterComponentFunc
}

type ImposerFunc = func(*ImposedParams)
