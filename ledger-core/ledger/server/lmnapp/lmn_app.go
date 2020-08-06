// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lmnapp

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insconveyor"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
)

func NewLightMaterialAppCompartment(appDeps injector.DependencyRegistry) *insconveyor.AppCompartment {
	return insconveyor.NewAppCompartment("LMN", appDeps, nil, nil,
		func(ctx context.Context, inject injector.DependencyInjector) insconveyor.AppCompartmentSetup {
			return insconveyor.AppCompartmentSetup{
				EventFactory:     nil,
				ConveyorCycleFn:  nil,
				ComponentSetupFn: nil,
			}
		})
}
