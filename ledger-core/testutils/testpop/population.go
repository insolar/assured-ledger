// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package testpop

import (
	"time"

	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/adapters"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/censusimpl"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

//nolint:interfacer
func CreateOneNodePopulationMock(t minimock.Tester, localRef reference.Global) census.OnlinePopulation {
	localNode := node.GenerateShortID(localRef)
	cp := profiles.NewCandidateProfileMock(t)
	cp.GetBriefIntroSignedDigestMock.Return(cryptkit.SignedDigest{})
	cp.GetDefaultEndpointMock.Return(adapters.NewOutbound("127.0.0.1:1"))
	cp.GetExtraEndpointsMock.Return(nil)
	cp.GetIssuedAtPulseMock.Return(pulse.MinTimePulse)
	cp.GetIssuedAtTimeMock.Return(time.Now())
	cp.GetIssuerIDMock.Return(localNode)
	cp.GetIssuerSignatureMock.Return(cryptkit.Signature{})
	cp.GetNodePublicKeyMock.Return(cryptkit.NewSignatureKeyHolderMock(t))
	cp.GetPowerLevelsMock.Return(member.PowerSet{0, 0, 0, 1})
	cp.GetPrimaryRoleMock.Return(member.PrimaryRoleVirtual)
	cp.GetReferenceMock.Return(localRef)
	cp.GetSpecialRolesMock.Return(0)
	cp.GetStartPowerMock.Return(1)
	cp.GetStaticNodeIDMock.Return(localNode)

	svf := cryptkit.NewSignatureVerifierFactoryMock(t)
	svf.CreateSignatureVerifierWithPKSMock.Return(nil)

	np := profiles.NewStaticProfileByFull(cp, nil)
	op := censusimpl.NewManyNodePopulation([]profiles.StaticProfile{np}, localNode, svf)

	// cs := census.NewActiveMock(t)
	// cs.GetOnlinePopulationMock.Return(&op)
	return &op
}

