// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nodestorage

import (
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
)

func TestNodeStorage_All(t *testing.T) {
	var all []rms.Node
	f := fuzz.New().Funcs(func(e *rms.Node, c fuzz.Continue) {
		e.ID.Set(gen.UniqueGlobalRef())
	})
	f.NumElements(5, 10).NilChance(0).Fuzz(&all)
	pulse := gen.PulseNumber()

	t.Run("returns correct nodes", func(t *testing.T) {
		nodeStorage := NewStorage()
		nodeStorage.nodes[pulse] = all
		result, err := nodeStorage.All(pulse)
		assert.NoError(t, err)
		assert.Equal(t, all, result)
	})

	t.Run("returns nil when empty nodes", func(t *testing.T) {
		nodeStorage := NewStorage()
		nodeStorage.nodes[pulse] = nil
		result, err := nodeStorage.All(pulse)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error when no nodes", func(t *testing.T) {
		nodeStorage := NewStorage()
		result, err := nodeStorage.All(pulse)
		assert.Equal(t, ErrNoNodes, err)
		assert.Nil(t, result)
	})
}

func TestNodeStorage_InRole(t *testing.T) {
	var (
		virtuals  []rms.Node
		materials []rms.Node
		all       []rms.Node
	)
	{
		f := fuzz.New().Funcs(func(e *rms.Node, c fuzz.Continue) {
			e.ID.Set(gen.UniqueGlobalRef())
			e.Role = member.StaticRoleVirtual
		})
		f.NumElements(5, 10).NilChance(0).Fuzz(&virtuals)
	}
	{
		f := fuzz.New().Funcs(func(e *rms.Node, c fuzz.Continue) {
			e.ID.Set(gen.UniqueGlobalRef())
			e.Role = member.StaticRoleLightMaterial
		})
		f.NumElements(5, 10).NilChance(0).Fuzz(&materials)
	}
	all = append(virtuals, materials...)
	pulse := gen.PulseNumber()

	t.Run("returns correct nodes", func(t *testing.T) {
		nodeStorage := NewStorage()
		nodeStorage.nodes[pulse] = all
		{
			result, err := nodeStorage.InRole(pulse, member.StaticRoleVirtual)
			assert.NoError(t, err)
			assert.Equal(t, virtuals, result)
		}
		{
			result, err := nodeStorage.InRole(pulse, member.StaticRoleLightMaterial)
			assert.NoError(t, err)
			assert.Equal(t, materials, result)
		}
	})

	t.Run("returns nil when empty nodes", func(t *testing.T) {
		nodeStorage := NewStorage()
		nodeStorage.nodes[pulse] = nil
		result, err := nodeStorage.InRole(pulse, member.StaticRoleVirtual)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error when no nodes", func(t *testing.T) {
		nodeStorage := NewStorage()
		result, err := nodeStorage.InRole(pulse, member.StaticRoleVirtual)
		assert.Equal(t, ErrNoNodes, err)
		assert.Nil(t, result)
	})
}

func TestStorage_Set(t *testing.T) {
	var nodes []rms.Node
	f := fuzz.New().Funcs(func(e *rms.Node, c fuzz.Continue) {
		e.ID.Set(gen.UniqueGlobalRef())
	})
	f.NumElements(5, 10).NilChance(0).Fuzz(&nodes)
	pulse := gen.PulseNumber()

	t.Run("saves correct nodes", func(t *testing.T) {
		nodeStorage := NewStorage()
		err := nodeStorage.Set(pulse, nodes)
		assert.NoError(t, err)
		assert.Equal(t, nodes, nodeStorage.nodes[pulse])
	})

	t.Run("saves nil if empty nodes", func(t *testing.T) {
		nodeStorage := NewStorage()
		err := nodeStorage.Set(pulse, []rms.Node{})
		assert.NoError(t, err)
		assert.Nil(t, nodeStorage.nodes[pulse])
	})

	t.Run("returns error when saving with the same pulse", func(t *testing.T) {
		nodeStorage := NewStorage()
		_ = nodeStorage.Set(pulse, nodes)
		err := nodeStorage.Set(pulse, nodes)
		assert.Equal(t, ErrOverride, err)
		assert.Equal(t, nodes, nodeStorage.nodes[pulse])
	})
}
