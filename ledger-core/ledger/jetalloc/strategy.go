// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package jetalloc

import (
	"crypto/sha256"
	"encoding/binary"
	"hash"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type MaterialAllocationStrategy interface {
	CreateCalculator(entropy longbits.FixedReader, population census.OnlinePopulation) MaterialAllocationCalculator
}

type MaterialAllocationCalculator interface {
	AllocationOfLine(local reference.Local) jet.Prefix

	// AllocationOfJets MUST receive list of all jets, otherwise allocation may be incorrect.
	// This requirements for all jets is intended to support balanced distribution algorithms.
	AllocationOfJets([]jet.ExactID, pulse.Number) map[jet.ID]node.ShortNodeID
}

type HashFactoryFunc = func() hash.Hash

func NewMaterialAllocationStrategy(usePower bool) MaterialAllocationStrategy {
	return defaultMaterialAllocationStrategy{sha256.New, usePower}
}

type defaultMaterialAllocationStrategy struct {
	hashFactory HashFactoryFunc
	usePower bool
}

func (v defaultMaterialAllocationStrategy) CreateCalculator(entropy longbits.FixedReader, population census.OnlinePopulation) MaterialAllocationCalculator {
	mPop := population.GetRolePopulation(member.PrimaryRoleLightMaterial)
	switch {
	case v.hashFactory == nil:
		panic(throw.IllegalState())
	case entropy == nil:
		panic(throw.IllegalValue())
	case mPop == nil || !mPop.IsValid():
		panic(throw.IllegalValue())
	case v.usePower:
		if mPop.GetWorkingPower() == 0 {
			panic(throw.IllegalValue())
		}
		return defaultMaterialAllocationCalc{ v.hashFactory, entropy, mPop.GetAssignmentByPower }

	default:
		if mPop.GetWorkingCount() <= 0 {
			panic(throw.IllegalValue())
		}
		return defaultMaterialAllocationCalc{ v.hashFactory, entropy, mPop.GetAssignmentByCount }
	}
}

type defaultMaterialAllocationCalc struct {
	hashFactory HashFactoryFunc
	entropy longbits.FixedReader
	assignFn census.AssignmentFunc
}

func (v defaultMaterialAllocationCalc) AllocationOfLine(local reference.Local) jet.Prefix {
	b := local.AsBytes()
	return NewPrefixCalc().FromSlice(jet.IDBitLen, b)
}

func (v defaultMaterialAllocationCalc) metricOfDrop(jt jet.DropID) uint64 {
	encoding := binary.LittleEndian

	h := v.hashFactory()
	var b [8]byte
	encoding.PutUint64(b[:], uint64(jt.ID()))
	_, _ = h.Write(b[:])
	_, _ = v.entropy.WriteTo(h)
	return longbits.CutOutUint64(h.Sum(nil))
}

func (v defaultMaterialAllocationCalc) AllocationOfJets(jets []jet.ExactID, pn pulse.Number) map[jet.ID]node.ShortNodeID {
	// TODO implement a balanced allocation function
	result := map[jet.ID]node.ShortNodeID{}
	for _, jt := range jets {
		jid := jt.ID()
		metric := v.metricOfDrop(jid.AsDrop(pn))
		nd, _ := v.assignFn(metric, 0)
		result[jid] = nd.GetNodeID()
	}
	return result
}

