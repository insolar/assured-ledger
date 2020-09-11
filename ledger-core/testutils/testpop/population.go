// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package testpop

import (
	"sort"
	"time"

	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/censusimpl"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

// CreateOneNodePopulationMock creates a NON-POWERED population
//nolint:interfacer
func CreateOneNodePopulationMock(t minimock.Tester, localRef reference.Global, role member.PrimaryRole) census.OnlinePopulation {
	localNode := node.GenerateShortID(localRef)
	np := createStaticProfile(t, localNode, localRef, role)

	svf := cryptkit.NewSignatureVerifierFactoryMock(t)
	svf.CreateSignatureVerifierWithPKSMock.Return(nil)
	op := censusimpl.NewManyNodePopulation([]profiles.StaticProfile{np}, localNode, svf)

	return &op
}

// CreateManyNodePopulationMock creates a POWERED population with all nodes of same role and power
func CreateManyNodePopulationMock(t minimock.Tester, count int, role member.PrimaryRole) census.OnlinePopulation {
	pr := make([]profiles.StaticProfile, 0, count)

	id := node.ShortNodeID(0)
	for ;count > 0; count-- {
		id++
		sp := createStaticProfile(t, id, reference.NewSelf(gen.UniqueLocalRef()), role)
		pr = append(pr, sp)
	}

	svf := cryptkit.NewSignatureVerifierFactoryMock(t)
	svf.CreateSignatureVerifierWithPKSMock.Return(nil)

	op := censusimpl.NewManyNodePopulation(pr, id, svf)

	// Following processing mimics consensus behavior on node ordering.
	// Ordering rules must be followed strictly.
	// WARNING! This code will only work properly when all nodes have same PrimaryRole and non-zero power

	dp := censusimpl.NewDynamicPopulationCopySelf(&op)
	for _, np := range op.GetProfiles() {
		if np.GetNodeID() != id  {
			dp.AddProfile(np.GetStatic())
		}
	}

	pfs := dp.GetUnorderedProfiles()
	for _, np := range pfs {
		np.SetOpMode(member.ModeNormal)
		pw := np.GetStatic().GetStartPower()
		np.SetPower(pw)
	}

	sort.Slice(pfs, func(i, j int) bool {
		// Power sorting is REVERSED
		return pfs[j].GetDeclaredPower() < pfs[i].GetDeclaredPower()
	})

	idx := member.AsIndex(0)
	for _, np := range pfs {
		np.SetIndex(idx)
		idx++
	}

	ap, _ := dp.CopyAndSeparate(false, nil)
	return ap
}

func createStaticProfile(t minimock.Tester, id node.ShortNodeID, ref reference.Global, role member.PrimaryRole) profiles.StaticProfile {
	cp := profiles.NewCandidateProfileMock(t)
	cp.GetBriefIntroSignedDigestMock.Return(cryptkit.SignedDigest{})
	cp.GetDefaultEndpointMock.Return(nil)
	cp.GetExtraEndpointsMock.Return(nil)
	cp.GetIssuedAtPulseMock.Return(pulse.MinTimePulse)
	cp.GetIssuedAtTimeMock.Return(time.Now())
	cp.GetIssuerIDMock.Return(id)
	cp.GetIssuerSignatureMock.Return(cryptkit.Signature{})
	cp.GetNodePublicKeyMock.Return(cryptkit.NewSigningKeyHolderMock(t))
	cp.GetPowerLevelsMock.Return(member.PowerSet{0, 0, 0, 255})
	cp.GetPrimaryRoleMock.Return(role)
	cp.GetReferenceMock.Return(ref)
	cp.GetSpecialRolesMock.Return(0)
	cp.GetStartPowerMock.Return(member.PowerOf(uint16(id)))
	cp.GetStaticNodeIDMock.Return(id)

	return profiles.NewStaticProfileByFull(cp, nil)
}
