// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package jetcoordinator

import (
	"context"
	"fmt"
	"sort"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	insolarPulse "github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/utils"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/utils/entropy"
)

// Coordinator is responsible for all jet interactions
type Coordinator struct {
	PlatformCryptographyScheme insolar.PlatformCryptographyScheme `inject:""`

	PulseAccessor   insolarPulse.Accessor   `inject:""`
	PulseCalculator insolarPulse.Calculator `inject:""`

	Nodes node.Accessor `inject:""`

	lightChainLimit int
	originRef       insolar.Reference
}

// NewJetCoordinator creates new coordinator instance.
func NewJetCoordinator(lightChainLimit int, originRef insolar.Reference) *Coordinator {
	return &Coordinator{lightChainLimit: lightChainLimit, originRef: originRef}
}

// Hardcoded roles count for validation and execution
const (
	VirtualValidatorCount  = 3
	MaterialValidatorCount = 3

	VirtualExecutorCount  = 1
	MaterialExecutorCount = 1
)

// Me returns current node.
func (jc *Coordinator) Me() insolar.Reference {
	return jc.originRef
}

// QueryRole returns node refs responsible for role bound operations for given object and pulse.
func (jc *Coordinator) QueryRole(
	ctx context.Context,
	role insolar.DynamicRole,
	objID insolar.ID,
	pulseNumber insolar.PulseNumber,
) ([]insolar.Reference, error) {
	switch role {
	case insolar.DynamicRoleVirtualExecutor:
		n, err := jc.VirtualExecutorForObject(ctx, objID, pulseNumber)
		if err != nil {
			return nil, errors.Wrapf(err, "calc DynamicRoleVirtualExecutor for object %v failed", objID.String())
		}
		return []insolar.Reference{*n}, nil

	case insolar.DynamicRoleVirtualValidator:
		return jc.VirtualValidatorsForObject(ctx, objID, pulseNumber)
	}

	inslogger.FromContext(ctx).Panicf("unexpected role %v", role.String())
	return nil, nil
}

// VirtualExecutorForObject returns list of VEs for a provided pulse and objID
func (jc *Coordinator) VirtualExecutorForObject(
	ctx context.Context, objID insolar.ID, pulse insolar.PulseNumber,
) (*insolar.Reference, error) {
	nodes, err := jc.virtualsForObject(ctx, objID, pulse, VirtualExecutorCount)
	if err != nil {
		return nil, err
	}
	return &nodes[0], nil
}

// VirtualValidatorsForObject returns list of VVs for a provided pulse and objID
func (jc *Coordinator) VirtualValidatorsForObject(
	ctx context.Context, objID insolar.ID, pulse insolar.PulseNumber,
) ([]insolar.Reference, error) {
	nodes, err := jc.virtualsForObject(ctx, objID, pulse, VirtualValidatorCount+VirtualExecutorCount)
	if err != nil {
		return nil, errors.Wrapf(err, "calc VirtualValidatorsForObject for object %v failed", objID.String())
	}
	// Skipping `VirtualExecutorCount` for validators
	// because it will be selected as the executor(s) for the same pulse.
	return nodes[VirtualExecutorCount:], nil
}

// LightExecutorForJet returns list of LEs for a provided pulse and jetID
func (jc *Coordinator) LightExecutorForJet(
	ctx context.Context, jetID insolar.ID, pulse insolar.PulseNumber,
) (*insolar.Reference, error) {
	nodes, err := jc.lightMaterialsForJet(ctx, jetID, pulse, MaterialExecutorCount)
	if err != nil {
		return nil, err
	}
	return &nodes[0], nil
}

// LightValidatorsForJet returns list of LVs for a provided pulse and jetID
func (jc *Coordinator) LightValidatorsForJet(
	ctx context.Context, jetID insolar.ID, pulse insolar.PulseNumber,
) ([]insolar.Reference, error) {
	nodes, err := jc.lightMaterialsForJet(ctx, jetID, pulse, MaterialValidatorCount+MaterialExecutorCount)
	if err != nil {
		return nil, err
	}
	// Skipping `MaterialExecutorCount` for validators
	// because it will be selected as the executor(s) for the same pulse.
	return nodes[MaterialExecutorCount:], nil
}

// IsBeyondLimit calculates if target pulse is behind clean-up limit
// or if currentPN|targetPN didn't found in in-memory pulse-storage.
func (jc *Coordinator) IsBeyondLimit(ctx context.Context, targetPN insolar.PulseNumber) (bool, error) {
	// Genesis case. When there is no any data on a lme
	if targetPN <= insolar.GenesisPulse.PulseNumber {
		return true, nil
	}

	latest, err := jc.PulseAccessor.Latest(ctx)
	if err != nil {
		return false, errors.Wrap(err, "failed to fetch pulse")
	}

	// Out target on the latest pulse. It's within limit.
	if latest.PulseNumber <= targetPN {
		return false, nil
	}

	iter := latest.PulseNumber
	for i := 1; i <= jc.lightChainLimit; i++ {
		stepBack, err := jc.PulseCalculator.Backwards(ctx, latest.PulseNumber, i)
		// We could not reach our target and ran out of known pulses. It means it's beyond limit.
		if err == insolarPulse.ErrNotFound {
			return true, nil
		}
		if err != nil {
			return false, errors.Wrap(err, "failed to calculate pulse")
		}
		// We reached our target. It's within limit.
		if iter <= targetPN {
			return false, nil
		}

		iter = stepBack.PulseNumber
	}
	// We iterated limit back. It means our data is further back and beyond limit.
	return true, nil
}

func (jc *Coordinator) virtualsForObject(
	ctx context.Context, objID insolar.ID, pulse insolar.PulseNumber, count int,
) ([]insolar.Reference, error) {
	candidates, err := jc.Nodes.InRole(pulse, insolar.StaticRoleVirtual)
	if err == node.ErrNoNodes {
		return nil, err
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch active virtual nodes for pulse %v", pulse)
	}
	if len(candidates) == 0 {
		return nil, errors.New(fmt.Sprintf("no active virtual nodes for pulse %d", pulse))
	}

	ent, err := jc.entropy(ctx, pulse)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch entropy for pulse %v", pulse)
	}

	return getRefs(
		jc.PlatformCryptographyScheme,
		utils.CircleXOR(ent[:], objID.Hash()),
		candidates,
		count,
	)
}

func (jc *Coordinator) lightMaterialsForJet(
	ctx context.Context, jetID insolar.ID, pulse insolar.PulseNumber, count int,
) ([]insolar.Reference, error) {
	prefix := insolar.JetID(jetID).Prefix()

	candidates, err := jc.Nodes.InRole(pulse, insolar.StaticRoleLightMaterial)
	if err == node.ErrNoNodes {
		return nil, err
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch active light nodes for pulse %v", pulse)
	}
	if len(candidates) == 0 {
		return nil, node.ErrNoNodes
	}

	ent, err := jc.entropy(ctx, pulse)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch entropy for pulse %v", pulse)
	}

	return getRefs(
		jc.PlatformCryptographyScheme,
		utils.CircleXOR(ent[:], prefix),
		candidates,
		count,
	)
}

func (jc *Coordinator) entropy(ctx context.Context, pulse insolar.PulseNumber) (insolar.Entropy, error) {
	current, err := jc.PulseAccessor.Latest(ctx)
	if err != nil {
		return insolar.Entropy{}, errors.Wrap(err, "failed to get current pulse")
	}

	if current.PulseNumber == pulse {
		return current.Entropy, nil
	}

	older, err := jc.PulseAccessor.ForPulseNumber(ctx, pulse)
	if err != nil {
		return insolar.Entropy{}, errors.Wrapf(err, "failed to fetch pulse data for pulse %v", pulse)
	}

	return older.Entropy, nil
}

func getRefs(
	scheme insolar.PlatformCryptographyScheme,
	e []byte,
	values []insolar.Node,
	count int,
) ([]insolar.Reference, error) {
	sort.SliceStable(values, func(i, j int) bool {
		return values[i].ID.Compare(values[j].ID) < 0
	})
	in := make([]interface{}, 0, len(values))
	for _, value := range values {
		in = append(in, interface{}(value.ID))
	}

	res, err := entropy.SelectByEntropy(scheme, e, in, count)
	if err != nil {
		return nil, err
	}
	out := make([]insolar.Reference, 0, len(res))
	for _, value := range res {
		out = append(out, value.(insolar.Reference))
	}
	return out, nil
}
