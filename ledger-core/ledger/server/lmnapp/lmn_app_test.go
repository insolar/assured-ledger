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

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/managed"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insconveyor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/testpop"
)

func TestStartCtm(t *testing.T) {
	instestlogger.SetTestOutput(t)

	// Reuse default compartment construction logic
	app := NewAppCompartment(configuration.Ledger{}, insapp.AppComponents{
		// Provide mocks for application-level / cross-compartment dependencies here
		MessageSender: messagesender.NewServiceMock(t),
	})
	require.NotNil(t, app)

	app.SetImposer(func(params *insconveyor.ImposedParams) {
		// impose upon default behavior

		// params.CompartmentSetup.ConveyorConfig allows to adjust conveyor's settings
		params.CompartmentSetup.ConveyorConfig.EventlessSleep = 0

		//	params.CompartmentSetup.EventFactoryFn - override event factory
		//	params.CompartmentSetup.ConveyorCycleFn - set cycle fn to control worker's stepping behavior

		// params.CompartmentSetup.Components - to add/remove/replace conveyor-depended components

		// params.CompartmentSetup.Dependencies - to add/remove/replace dependencies provided to SMs
		params.CompartmentSetup.Dependencies.DeleteDependency("something")

		// params.ComponentInterceptFn can be used to prevent or to replace any of
		// conveyor-depended components.
		// For now this is similar to do changes for params.CompartmentSetup.Components
		params.ComponentInterceptFn = func(c managed.Component) managed.Component {
			return c
		}
	})

	ctx := context.Background()

	err := app.Init(ctx)
	require.NoError(t, err)

	err = app.Start(ctx)
	require.NoError(t, err)

	localRef := gen.UniqueGlobalRef()
	pg := testutils.NewPulseGenerator(10, testpop.CreateOneNodePopulationMock(t, localRef))
	pg.Generate()

	bd := app.GetBeatDispatcher()
	for i := 3; i > 0; i-- {
		time.Sleep(10*time.Millisecond)
		pg.Generate()

		ack, _ := beat.NewAck(func(beat.AckData) {})
		bd.PrepareBeat(ack)
		bd.CommitBeat(pg.GetLastBeat())
	}

	err = app.Stop(ctx)
	require.NoError(t, err)
}
