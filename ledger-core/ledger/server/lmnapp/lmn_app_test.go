// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lmnapp

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/managed"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insconveyor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
)

func TestStartCtm(t *testing.T) {
	instestlogger.SetTestOutput(t)
	app := NewAppCompartment(configuration.Ledger{}, insapp.AppComponents{})
	require.NotNil(t, app)

	app.SetImposer(func(params *insconveyor.ImposedParams) {
		// impose upon default behavior

		params.ComponentInterceptFn = func(c managed.Component) managed.Component {
			// switch c.(type) {
			// case InterceptedType:
			// }
			return c
		}

		params.ConveyorRegistry.ScanDependencies(func(id string, v interface{}) bool {
			if id == "something" {
				params.ConveyorRegistry.DeleteDependency(id)
			}

			return false
		})
	})

	ctx := context.Background()

	err := app.Init(ctx)
	require.NoError(t, err)

	err = app.Start(ctx)
	require.NoError(t, err)

	time.Sleep(time.Millisecond)

	err = app.Stop(ctx)
	require.NoError(t, err)
}
