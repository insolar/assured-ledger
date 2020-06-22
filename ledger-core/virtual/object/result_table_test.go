// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package object

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

func TestResultTable(t *testing.T) {
	rt := NewResultTable()

	require.Equal(t, 0, len(rt.GetList(contract.CallIntolerable).results))
	require.Equal(t, 0, len(rt.GetList(contract.CallTolerable).results))

	pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
	currentPulse := pd.PulseNumber
	object := gen.UniqueIDWithPulse(currentPulse)
	ref := reference.NewSelf(object)

	expResult1 := payload.VCallResult{
		CallType:                0,
		CallFlags:               0,
		CallAsOf:                0,
		Caller:                  gen.UniqueReference(),
		Callee:                  gen.UniqueReference(),
		ResultFlags:             nil,
		CallOutgoing:            gen.UniqueReference().GetLocal(),
		CallIncoming:            gen.UniqueReference().GetLocal(),
		PayloadHash:             nil,
		DelegationSpec:          payload.CallDelegationToken{},
		CallIncomingResult:      payload.LocalReference{},
		ProducerSignature:       nil,
		RegistrarSignature:      nil,
		RegistrarDelegationSpec: payload.CallDelegationToken{},
		EntryHeadHash:           nil,
		SecurityContext:         nil,
		ReturnArguments:         nil,
		ExtensionHashes:         nil,
		Extensions:              nil,
	}

	intolerableList := rt.GetList(contract.CallIntolerable)
	intolerableList.Add(ref, &expResult1)

	require.Equal(t, 1, len(rt.GetList(contract.CallIntolerable).results))
	require.Equal(t, 0, len(rt.GetList(contract.CallTolerable).results))

	aclResult1, _ := rt.GetList(contract.CallIntolerable).Get(ref)

	require.Equal(t, &expResult1, aclResult1)

	expResult2 := payload.VCallResult{
		CallType:                0,
		CallFlags:               0,
		CallAsOf:                0,
		Caller:                  gen.UniqueReference(),
		Callee:                  gen.UniqueReference(),
		ResultFlags:             nil,
		CallOutgoing:            gen.UniqueReference().GetLocal(),
		CallIncoming:            gen.UniqueReference().GetLocal(),
		PayloadHash:             nil,
		DelegationSpec:          payload.CallDelegationToken{},
		CallIncomingResult:      payload.LocalReference{},
		ProducerSignature:       nil,
		RegistrarSignature:      nil,
		RegistrarDelegationSpec: payload.CallDelegationToken{},
		EntryHeadHash:           nil,
		SecurityContext:         nil,
		ReturnArguments:         nil,
		ExtensionHashes:         nil,
		Extensions:              nil,
	}

	tolerableList := rt.GetList(contract.CallTolerable)
	tolerableList.Add(ref, &expResult2)

	require.Equal(t, 1, len(rt.GetList(contract.CallIntolerable).results))
	require.Equal(t, 1, len(rt.GetList(contract.CallTolerable).results))

	aclResult2, _ := rt.GetList(contract.CallTolerable).Get(ref)

	require.Equal(t, &expResult2, aclResult2)
}
