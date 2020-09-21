// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package affinity

import (
	"crypto/sha256"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

// Coordinator is responsible for all jet interactions
type Coordinator struct {
	PlatformCryptographyScheme cryptography.PlatformCryptographyScheme `inject:""`

	PulseAccessor beat.History `inject:""`

	originRef reference.Global
}

// NewAffinityHelper creates new Helper instance.
func NewAffinityHelper(originRef reference.Global) *Coordinator {
	return &Coordinator{originRef: originRef}
}

// Me returns current node.
func (jc *Coordinator) Me() reference.Global {
	return jc.originRef
}

// QueryRole returns node refs responsible for role bound operations for given object and pulse.
func (jc *Coordinator) QueryRole(role DynamicRole, objID reference.Holder, pn pulse.Number) ([]reference.Global, error) {
	var (
		nodeID reference.Global
		err    error
	)
	switch role {
	case DynamicRoleVirtualExecutor:
		nodeID, err = jc.VirtualExecutorForObject(objID, pn)
	case DynamicRoleLightExecutor:
		nodeID, err = jc.LightExecutorForObject(objID, pn)
	default:
		panic(throw.NotImplemented())
	}

	if err != nil {
		return nil, throw.WithDetails(err, struct {
			Ref reference.Holder
			PN  pulse.Number
		}{objID, pn})
	}
	return []reference.Global{nodeID}, nil
}

// VirtualExecutorForObject returns list of VEs for a provided pulse and objID
func (jc *Coordinator) VirtualExecutorForObject(objID reference.Holder, pn pulse.Number) (reference.Global, error) {
	pc, err := jc.PulseAccessor.TimeBeat(pn)
	switch {
	case err != nil:
		return reference.Global{}, err
	case pc.Online == nil:
		return reference.Global{}, throw.IllegalState()
	}

	role := pc.Online.GetRolePopulation(member.PrimaryRoleVirtual)
	if role == nil {
		return reference.Global{}, throw.E("role without nodes", struct {
			PrimaryRole member.PrimaryRole
			Population  census.OnlinePopulation
		}{
			member.PrimaryRoleVirtual,
			pc.Online,
		})
	}

	base := objID.GetBase()
	h := sha256.New()
	_, err = h.Write(base.AsBytes())
	if err != nil {
		return reference.Global{}, err
	}
	_, err = h.Write(pc.PulseEntropy[:])
	if err != nil {
		return reference.Global{}, err
	}
	metric := longbits.CutOutUint64(h.Sum(nil))

	assigned, _ := role.GetAssignmentByCount(metric, 0)
	if assigned == nil {
		return reference.Global{}, throw.E("unable to assign node of role", struct {
			PrimaryRole member.PrimaryRole
			Population  census.OnlinePopulation
		}{
			member.PrimaryRoleVirtual,
			pc.Online,
		})
	}
	ref := assigned.GetStatic().GetExtension().GetReference()

	return ref, nil
}

func (jc *Coordinator) LightExecutorForObject(objID reference.Holder, pn pulse.Number) (reference.Global, error) {
	pc, err := jc.PulseAccessor.TimeBeat(pn)
	switch {
	case err != nil:
		return reference.Global{}, err
	case pc.Online == nil:
		return reference.Global{}, throw.IllegalState()
	}

	role := pc.Online.GetRolePopulation(member.PrimaryRoleLightMaterial)
	if role != nil {
		panic(throw.Unsupported())
	}

	return jc.VirtualExecutorForObject(objID, pn)
}
