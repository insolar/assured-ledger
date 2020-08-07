// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insconveyor

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/managed"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
)

type EventFactory interface {
	InputEvent(context.Context, conveyor.InputEvent, conveyor.InputContext) (conveyor.InputSetup, error)
	SetupComponents(context.Context, injector.DependencyInjector, managed.RegisterComponentFunc)
}
