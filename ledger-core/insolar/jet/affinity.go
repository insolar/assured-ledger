// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package jet

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/appctl"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

// AffinityCoordinator is responsible for all jet interactions
type AffinityCoordinator struct {
	PlatformCryptographyScheme cryptography.PlatformCryptographyScheme `inject:""`

	PulseAccessor   appctl.Accessor      `inject:""`

	originRef       reference.Global
}

// NewAffinityHelper creates new AffinityHelper instance.
func NewAffinityHelper(originRef reference.Global) *AffinityCoordinator {
	return &AffinityCoordinator{originRef: originRef}
}

// Me returns current node.
func (jc *AffinityCoordinator) Me() reference.Global {
	return jc.originRef
}

// QueryRole returns node refs responsible for role bound operations for given object and pulse.
func (jc *AffinityCoordinator) QueryRole(
	ctx context.Context,
	role node.DynamicRole,
	objID reference.Holder,
	pn pulse.Number,
) ([]reference.Global, error) {
	if role == node.DynamicRoleVirtualExecutor {
		n, err := jc.VirtualExecutorForObject(ctx, objID, pn)
		if err != nil {
			return nil, throw.WithDetails(err, struct { Ref reference.Holder; PN pulse.Number } {objID, pn })
		}
		return []reference.Global{n}, nil
	}

	panic(throw.NotImplemented())
}

// VirtualExecutorForObject returns list of VEs for a provided pulse and objID
func (jc *AffinityCoordinator) VirtualExecutorForObject(
	ctx context.Context, objID reference.Holder, pn pulse.Number,
) (reference.Global, error) {
	pc, err := jc.PulseAccessor.ForPulseNumber(ctx, pn)
	switch {
	case err != nil:
		return reference.Global{}, err
	case pc.Online == nil:
		return reference.Global{}, throw.IllegalState()
	}

	role := pc.Online.GetRolePopulation(member.PrimaryRoleVirtual)
	if role == nil {
		return reference.Global{}, throw.E("role without nodes", struct {
			member.PrimaryRole
			census.OnlinePopulation
		} {
			member.PrimaryRoleVirtual,
			pc.Online,
		})
	}

	base := objID.GetBase()
	b := base.AsBytes()
	xorBytes(b, pc.PulseEntropy[:])
	metric := longbits.CutOutUint64(b)

	metric += uint64(base.Pulse())
	assigned, _ := role.GetAssignmentByCount(metric, 0)
	if assigned == nil {
		return reference.Global{}, throw.E("unable to assign node of role", struct {
			member.PrimaryRole
			census.OnlinePopulation
		}{
			member.PrimaryRoleVirtual,
			pc.Online,
		})
	}
	ref := assigned.GetStatic().GetExtension().GetReference()

	return ref, nil
}

func xorBytes(dest []byte, b []byte) {
	n := len(dest)
	for i := range b {
		dest[i % n] ^= b[i]
	}
}
