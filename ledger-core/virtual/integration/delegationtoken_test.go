// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"context"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func TestDelegationToken_CheckTokenField(t *testing.T) {
	t.Skip("https://insolar.atlassian.net/browse/PLAT-588")
	tests := []struct {
		name         string
		testRailID   string
		fakeCaller   bool
		fakeCallee   bool
		fakeOutgoing bool
	}{
		{
			name:         "Fail with wrong caller in token",
			testRailID:   "C5197",
			fakeCaller:   true,
			fakeCallee:   false,
			fakeOutgoing: false,
		},
		{
			name:         "Fail with wrong callee in token",
			testRailID:   "C5198",
			fakeCaller:   false,
			fakeCallee:   true,
			fakeOutgoing: false,
		},
		{
			name:         "Fail with wrong outgoing in token",
			testRailID:   "C5199",
			fakeCaller:   false,
			fakeCallee:   false,
			fakeOutgoing: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Log(test.testRailID)

			mc := minimock.NewController(t)

			server, ctx := utils.NewUninitializedServer(nil, t)
			defer server.Stop()

			runnerMock := logicless.NewServiceMock(ctx, mc, nil)
			server.ReplaceRunner(runnerMock)

			jetCoordinatorMock := jet.NewAffinityHelperMock(t)
			auth := authentication.NewService(ctx, jetCoordinatorMock)
			server.ReplaceAuthenticationService(auth)

			var errorFound bool
			logHandler := func(arg interface{}) {
				if err, ok := arg.(error); ok {
					if severity, sok := throw.GetSeverity(err); sok && severity == throw.RemoteBreachSeverity {
						errorFound = true
					}
				}
			}
			logger := utils.InterceptLog(inslogger.FromContext(ctx), logHandler)
			server.OverrideConveyorFactoryLogContext(inslogger.SetLogger(ctx, logger))

			server.Init(ctx)

			jetCaller := server.RandomGlobalWithPulse()
			jetCoordinatorMock.
				MeMock.Return(jetCaller).
				QueryRoleMock.Set(
				func(_ context.Context, _ node.DynamicRole, obj reference.Local, pulse pulse.Number) ([]reference.Global, error) {
					return []reference.Global{jetCaller}, nil
				})

			var (
				isolation = contract.ConstructorIsolation()
				callFlags = payload.BuildCallFlags(isolation.Interference, isolation.State)

				class    = gen.UniqueGlobalRef()
				outgoing = server.BuildRandomOutgoingWithPulse()

				constructorPulse = server.GetPulse().PulseNumber

				delegationToken payload.CallDelegationToken
			)

			delegationToken = server.DelegationToken(reference.NewRecordOf(class, outgoing.GetLocal()), server.GlobalCaller(), outgoing)
			switch {
			case test.fakeCaller:
				delegationToken.Callee = server.RandomGlobalWithPulse()
			case test.fakeCallee:
				delegationToken.Callee = server.RandomGlobalWithPulse()
			case test.fakeOutgoing:
				delegationToken.Outgoing = server.RandomGlobalWithPulse()
			}

			pl := payload.VCallRequest{
				CallType:       payload.CTConstructor,
				CallFlags:      callFlags,
				CallAsOf:       constructorPulse,
				Callee:         class,
				CallSiteMethod: "New",
				CallOutgoing:   outgoing,
				DelegationSpec: delegationToken,
			}
			server.SendPayload(ctx, &pl)
			server.WaitIdleConveyor()

			assert.True(t, errorFound)

			mc.Finish()
		})
	}
}
