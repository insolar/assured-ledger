// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package jepsen_tests

import (
	"context"
	"testing"

	"github.com/google/gops/agent"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/application/testutils"
	"github.com/insolar/assured-ledger/ledger-core/application/testutils/launchnet"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func init() {
	// starts gops agent https://github.com/google/gops on default addr (127.0.0.1:0)
	if err := agent.Listen(agent.Options{}); err != nil {
		panic(err)
	}
}

func TestMultiNode_FakeNetwork(t *testing.T) {
	instestlogger.SetTestOutputWithStub()

	cloudRunner := launchnet.PrepareCloudRunner(
		launchnet.WithRunning(5, 0, 0),
		launchnet.WithPrepared(6, 0, 0),
		launchnet.WithCloudFileLogging())

	assert.NoError(t, cloudRunner.ControlledRun(
		func(helper *launchnet.Helper) error {

			testutils.SetAPIPortsByAddresses(helper.GetApiAddresses())

			ctx := context.Background()
			for i := 0; i < 100; i++ {
				_, err := testutils.CreateSimpleWallet(ctx)
				if err != nil {
					return throw.W(err, "failed to create wallet")
				}
			}

			newNode := helper.AddNode(member.PrimaryRoleVirtual)

			assert.False(t, newNode.IsZero())

			helper.IncrementPulse(ctx)

			testutils.SetAPIPortsByAddresses(helper.GetApiAddresses())

			for i := 0; i < 100; i++ {
				_, err := testutils.CreateSimpleWallet(ctx)
				if err != nil {
					return throw.W(err, "failed to create wallet")
				}
			}

			helper.StopNode(member.PrimaryRoleVirtual, newNode)

			helper.IncrementPulse(ctx)

			testutils.SetAPIPortsByAddresses(helper.GetApiAddresses())

			for i := 0; i < 100; i++ {
				_, err := testutils.CreateSimpleWallet(ctx)
				if err != nil {
					return throw.W(err, "failed to create wallet")
				}
			}

			return nil
		},
	))
}
