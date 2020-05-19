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

	"github.com/insolar/assured-ledger/ledger-core/v2/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/nodestorage"
	insolarPulse "github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/entropy"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

// Coordinator is responsible for all jet interactions
type Coordinator struct {
	PlatformCryptographyScheme cryptography.PlatformCryptographyScheme `inject:""`

	PulseAccessor   insolarPulse.Accessor   `inject:""`
	PulseCalculator insolarPulse.Calculator `inject:""`

	Nodes nodestorage.Accessor `inject:""`

	lightChainLimit int
	originRef       reference.Global
}

// NewJetCoordinator creates new coordinator instance.
func NewJetCoordinator(lightChainLimit int, originRef reference.Global) *Coordinator {
	return &Coordinator{lightChainLimit: lightChainLimit, originRef: originRef}
}

// Hardcoded roles count for validation and execution
const (
	VirtualExecutorCount  = 1
)

// Me returns current node.
func (jc *Coordinator) Me() reference.Global {
	return jc.originRef
}

// QueryRole returns node refs responsible for role bound operations for given object and pulse.
func (jc *Coordinator) QueryRole(
	ctx context.Context,
	role node.DynamicRole,
	objID reference.Local,
	pulseNumber pulse.Number,
) ([]reference.Global, error) {
	if role == node.DynamicRoleVirtualExecutor {
		n, err := jc.VirtualExecutorForObject(ctx, objID, pulseNumber)
		if err != nil {
			return nil, errors.Wrapf(err, "calc DynamicRoleVirtualExecutor for object %v failed", objID.String())
		}
		return []reference.Global{n}, nil
	}

	inslogger.FromContext(ctx).Panicf("unexpected role %v", role.String())
	return nil, nil
}

// VirtualExecutorForObject returns list of VEs for a provided pulse and objID
func (jc *Coordinator) VirtualExecutorForObject(
	ctx context.Context, objID reference.Local, pulse pulse.Number,
) (reference.Global, error) {
	nodes, err := jc.virtualsForObject(ctx, objID, pulse, VirtualExecutorCount)
	if err != nil {
		return reference.Global{}, err
	}
	return nodes[0], nil
}

// IsBeyondLimit calculates if target pulse is behind clean-up limit
// or if currentPN|targetPN didn't found in in-memory pulse-storage.
func (jc *Coordinator) IsBeyondLimit(ctx context.Context, targetPN pulse.Number) (bool, error) {
	// Genesis case. When there is no any data on a lme
	if targetPN <= insolarPulse.GenesisPulse.PulseNumber {
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
	ctx context.Context, objID reference.Local, pulse pulse.Number, count int,
) ([]reference.Global, error) {
	candidates, err := jc.Nodes.InRole(pulse, node.StaticRoleVirtual)
	if err == nodestorage.ErrNoNodes {
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

func (jc *Coordinator) entropy(ctx context.Context, pulse pulse.Number) (insolarPulse.Entropy, error) {
	current, err := jc.PulseAccessor.Latest(ctx)
	if err != nil {
		return insolarPulse.Entropy{}, errors.Wrap(err, "failed to get current pulse")
	}

	if current.PulseNumber == pulse {
		return current.Entropy, nil
	}

	older, err := jc.PulseAccessor.ForPulseNumber(ctx, pulse)
	if err != nil {
		return insolarPulse.Entropy{}, errors.Wrapf(err, "failed to fetch pulse data for pulse %v", pulse)
	}

	return older.Entropy, nil
}

func getRefs(
	scheme cryptography.PlatformCryptographyScheme,
	e []byte,
	values []node.Node,
	count int,
) ([]reference.Global, error) {
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
	out := make([]reference.Global, 0, len(res))
	for _, value := range res {
		out = append(out, value.(reference.Global))
	}
	return out, nil
}
