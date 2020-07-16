// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package jet

import (
	"context"
	"sort"

	"github.com/insolar/assured-ledger/ledger-core/appctl"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/nodestorage"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/entropy"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

// AffinityCoordinator is responsible for all jet interactions
type AffinityCoordinator struct {
	PlatformCryptographyScheme cryptography.PlatformCryptographyScheme `inject:""`

	PulseAccessor   appctl.Accessor      `inject:""`

	Nodes nodestorage.Accessor `inject:""`

	lightChainLimit int
	originRef       reference.Global
}

// NewAffinityHelper creates new AffinityHelper instance.
func NewAffinityHelper(lightChainLimit int, originRef reference.Global) *AffinityCoordinator {
	return &AffinityCoordinator{lightChainLimit: lightChainLimit, originRef: originRef}
}

// Hardcoded roles count for validation and execution
const (
	VirtualExecutorCount = 1
)

// Me returns current node.
func (jc *AffinityCoordinator) Me() reference.Global {
	return jc.originRef
}

// QueryRole returns node refs responsible for role bound operations for given object and pulse.
func (jc *AffinityCoordinator) QueryRole(
	ctx context.Context,
	role node.DynamicRole,
	objID reference.Local,
	pulseNumber pulse.Number,
) ([]reference.Global, error) {
	if role == node.DynamicRoleVirtualExecutor {
		n, err := jc.VirtualExecutorForObject(ctx, objID, pulseNumber)
		if err != nil {
			return nil, throw.W(err, "calc DynamicRoleVirtualExecutor for object %v failed", objID.String())
		}
		return []reference.Global{n}, nil
	}

	inslogger.FromContext(ctx).Panicf("unexpected role %v", role.String())
	return nil, nil
}

// VirtualExecutorForObject returns list of VEs for a provided pulse and objID
func (jc *AffinityCoordinator) VirtualExecutorForObject(
	ctx context.Context, objID reference.Local, pulse pulse.Number,
) (reference.Global, error) {
	nodes, err := jc.virtualsForObject(ctx, objID, pulse, VirtualExecutorCount)
	if err != nil {
		return reference.Global{}, err
	}
	return nodes[0], nil
}

func (jc *AffinityCoordinator) virtualsForObject(
	ctx context.Context, objID reference.Local, pn pulse.Number, count int,
) ([]reference.Global, error) {
	candidates, err := jc.Nodes.InRole(pn, member.StaticRoleVirtual)
	if err == nodestorage.ErrNoNodes {
		return nil, err
	}
	if err != nil {
		return nil, throw.W(err, "failed to fetch active virtual nodes for pulse", struct {
			Pulse pulse.Number
		}{
			Pulse: pn,
		})
	}
	if len(candidates) == 0 {
		return nil, throw.E("no active virtual nodes for pulse", struct {
			Pulse pulse.Number
		}{
			Pulse: pn,
		})
	}

	ent, err := jc.entropy(ctx, pn)
	if err != nil {
		return nil, throw.W(err, "failed to fetch entropy for pulse", struct {
			Pulse pulse.Number
		}{
			Pulse: pn,
		})
	}

	return getRefs(
		jc.PlatformCryptographyScheme,
		CircleXOR(ent[:], objID.IdentityHashBytes()),
		candidates,
		count,
	)
}

// CircleXOR performs XOR for 'value' and 'src'. The result is returned as new byte slice.
// If 'value' is smaller than 'dst', XOR starts from the beginning of 'src'.
func CircleXOR(value, src []byte) []byte {
	result := make([]byte, len(value))
	srcLen := len(src)
	for i := range result {
		result[i] = value[i] ^ src[i%srcLen]
	}
	return result
}

func (jc *AffinityCoordinator) entropy(ctx context.Context, pn pulse.Number) (pulsestor.Entropy, error) {
	pc, err := jc.PulseAccessor.Latest(ctx)
	if err != nil {
		return pulsestor.Entropy{}, throw.W(err, "failed to get current pulse")
	}

	if pc.PulseNumber != pn {
		pc, err = jc.PulseAccessor.ForPulseNumber(ctx, pn)
		if err != nil {
			return pulsestor.Entropy{}, throw.W(err, "failed to fetch pulse data for pulse", struct {
				Pulse pulse.Number
			}{
				Pulse: pn,
			})
		}
	}

	etp := pulsestor.Entropy{}
	copy(etp[:32], pc.PulseEntropy[:])
	copy(etp[32:], pc.PulseEntropy[:])

	return etp, nil
}

func getRefs(
	scheme cryptography.PlatformCryptographyScheme,
	e []byte,
	values []rms.Node,
	count int,
) ([]reference.Global, error) {

	in := make([]interface{}, len(values))
	for i, value := range values {
		in[i] = value.ID.GetGlobal()
	}

	sort.SliceStable(in, func(i, j int) bool {
		return in[i].(reference.Global).Compare(in[j].(reference.Global)) < 0
	})

	res, err := entropy.SelectByEntropy(scheme, e, in, count)
	if err != nil {
		return nil, err
	}
	out := make([]reference.Global, 0, len(res))
	for _, value := range res {
		out = append(out, value.(reference.Global))
	}
	return out, nil
}
