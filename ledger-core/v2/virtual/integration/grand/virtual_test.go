// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build slowtest

package grand

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/genesisrefs"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/reply"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/machinesmanager"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils"
)

func TestVirtual_BasicOperations(t *testing.T) {
	t.Parallel()

	cfg := DefaultVMConfig()

	t.Run("happy path", func(t *testing.T) {
		ctx := inslogger.TestContext(t)

		expectedRes := struct {
			blip string
		}{
			blip: "blop",
		}

		mle := testutils.NewMachineLogicExecutorMock(t).CallMethodMock.Set(
			func(_ context.Context, _ *insolar.LogicCallContext, _ insolar.Reference, _ []byte, _ string, _ insolar.Arguments) ([]byte, insolar.Arguments, error) {
				return insolar.MustSerialize(expectedRes), insolar.MustSerialize(expectedRes), nil
			},
		)

		mm := machinesmanager.NewMachinesManagerMock(t).GetExecutorMock.Set(
			func(_ insolar.MachineType) (insolar.MachineLogicExecutor, error) {
				return mle, nil
			},
		).RegisterExecutorMock.Set(
			func(_ insolar.MachineType, _ insolar.MachineLogicExecutor) error {
				return nil
			},
		)

		s, err := NewVirtualServer(t, ctx, cfg).SetMachinesManager(mm).PrepareAndStart()
		require.NoError(t, err)
		defer s.Stop(ctx)

		// Prepare environment (mimic) for first call
		var objectID *insolar.ID
		{
			codeID, err := s.mimic.AddCode(ctx, []byte{})
			require.NoError(t, err)

			prototypeID, err := s.mimic.AddObject(ctx, *codeID, true, []byte{})
			require.NoError(t, err)

			objectID, err = s.mimic.AddObject(ctx, *prototypeID, false, []byte{})
			require.NoError(t, err)
		}
		t.Logf("iniitialization done")

		objectRef := insolar.NewReference(*objectID)

		res, requestRef, err := CallContract(
			s, objectRef, "good.CallMethod", nil, s.pulseGenerator.GetLastPulse().PulseNumber,
		)
		require.NoError(t, err)

		assert.NotEmpty(t, requestRef)
		assert.Equal(t, &reply.CallMethod{
			Object: objectRef,
			Result: insolar.MustSerialize(expectedRes),
		}, res)
	})

	t.Run("create user test", func(t *testing.T) {
		ctx := inslogger.TestContext(t)

		s, err := NewVirtualServer(t, ctx, cfg).WithGenesis().PrepareAndStart()
		require.NoError(t, err)
		defer s.Stop(ctx)

		user, err := NewUserWithKeys()
		if err != nil {
			panic("failed to create new user: " + err.Error())
		}

		callMethodReply, _, err := s.BasicAPICall(ctx, "member.create", nil, genesisrefs.ContractRootMember, user)
		if err != nil {
			panic(err.Error())
		}

		var result map[string]interface{}
		if err := insolar.Deserialize(callMethodReply.(*reply.CallMethod).Result, &result); err != nil {
			panic(err.Error())
		}

		assert.Nil(t, result["Error"])
		assert.NotNil(t, result["Returns"].([]interface{})[0].(map[string]interface{})["reference"])
	})

	t.Run("create and transfer test", func(t *testing.T) {
		ctx := inslogger.TestContext(t)

		s, err := NewVirtualServer(t, ctx, cfg).WithGenesis().PrepareAndStart()
		require.NoError(t, err)
		defer s.Stop(ctx)

		user1, err := NewUserWithKeys()
		if err != nil {
			panic("failed to create new user: " + err.Error())
		}
		user2, err := NewUserWithKeys()
		if err != nil {
			panic("failed to create new user: " + err.Error())
		}

		var walletReference1 insolar.Reference
		{
			callMethodReply, _, err := s.BasicAPICall(ctx, "member.create", nil, genesisrefs.ContractRootMember, user1)
			if err != nil {
				panic(err.Error())
			}

			var result map[string]interface{}
			if err := insolar.Deserialize(callMethodReply.(*reply.CallMethod).Result, &result); err != nil {
				panic(err.Error())
			}

			assert.Nil(t, result["Error"])

			walletReferenceString := result["Returns"].([]interface{})[0].(map[string]interface{})["reference"]
			assert.NotNil(t, walletReferenceString)
			assert.IsType(t, "", walletReferenceString)

			walletReference, err := insolar.NewReferenceFromString(walletReferenceString.(string))
			assert.NoError(t, err)

			walletReference1 = *walletReference
		}

		var walletReference2 insolar.Reference
		{
			callMethodReply, _, err := s.BasicAPICall(ctx, "member.create", nil, genesisrefs.ContractRootMember, user2)
			if err != nil {
				panic(err.Error())
			}

			var result map[string]interface{}
			if err := insolar.Deserialize(callMethodReply.(*reply.CallMethod).Result, &result); err != nil {
				panic(err.Error())
			}

			assert.Nil(t, result["Error"])
			assert.NotNil(t, result["Returns"].([]interface{})[0].(map[string]interface{})["reference"])

			walletReferenceString := result["Returns"].([]interface{})[0].(map[string]interface{})["reference"]
			assert.NotNil(t, walletReferenceString)
			assert.IsType(t, "", walletReferenceString)

			walletReference, err := insolar.NewReferenceFromString(walletReferenceString.(string))
			assert.NoError(t, err)

			walletReference2 = *walletReference
		}

		var feeWalletBalance string
		{
			callParams := map[string]interface{}{"reference": FeeWalletUser.Reference.String()}
			callMethodReply, _, err := s.BasicAPICall(ctx, "member.getBalance", callParams, FeeWalletUser.Reference, FeeWalletUser)
			if err != nil {
				panic(err.Error())
			}

			var result map[string]interface{}
			if err := insolar.Deserialize(callMethodReply.(*reply.CallMethod).Result, &result); err != nil {
				panic(err.Error())
			}
			require.Nil(t, result["Error"])

			fmt.Printf("%#v\n", result)
			feeWalletBalance = result["Returns"].([]interface{})[0].(map[string]interface{})["balance"].(string)
		}

		{
			callParams := map[string]interface{}{"amount": "10000", "toMemberReference": walletReference2.String()}
			callMethodReply, _, err := s.BasicAPICall(ctx, "member.transfer", callParams, walletReference1, user1)
			if err != nil {
				panic(err.Error())
			}

			var result map[string]interface{}
			if err := insolar.Deserialize(callMethodReply.(*reply.CallMethod).Result, &result); err != nil {
				panic(err.Error())
			}

			assert.Nil(t, result["Error"])
		}

		{
			for i := 1; i < 30; i++ {
				callParams := map[string]interface{}{"reference": FeeWalletUser.Reference.String()}
				callMethodReply, _, err := s.BasicAPICall(ctx, "member.getBalance", callParams, FeeWalletUser.Reference, FeeWalletUser)
				if err != nil {
					panic(err.Error())
				}

				var result map[string]interface{}
				if err := insolar.Deserialize(callMethodReply.(*reply.CallMethod).Result, &result); err != nil {
					panic(err.Error())
				}
				require.Nil(t, result["Error"])

				fmt.Printf("%#v\n", result)
				newBalance := result["Returns"].([]interface{})[0].(map[string]interface{})["balance"].(string)

				if newBalance != feeWalletBalance {
					break
				}

				time.Sleep(100 * time.Millisecond)
				if i == 29 {
					assert.FailNow(t, "failed to wait money in feeWallet")
				}
			}
		}
	})
}
