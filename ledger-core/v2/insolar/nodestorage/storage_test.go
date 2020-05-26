// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nodestorage

import (
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/gen"
)

func TestNodeStorage_All(t *testing.T) {
	t.Parallel()

	var all []node.Node
	f := fuzz.New().Funcs(func(e *node.Node, c fuzz.Continue) {
		e.ID = gen.UniqueReference()
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
	t.Parallel()

	var (
		virtuals  []node.Node
		materials []node.Node
		all       []node.Node
	)
	{
		f := fuzz.New().Funcs(func(e *node.Node, c fuzz.Continue) {
			e.ID = gen.UniqueReference()
			e.Role = node.StaticRoleVirtual
		})
		f.NumElements(5, 10).NilChance(0).Fuzz(&virtuals)
	}
	{
		f := fuzz.New().Funcs(func(e *node.Node, c fuzz.Continue) {
			e.ID = gen.UniqueReference()
			e.Role = node.StaticRoleLightMaterial
		})
		f.NumElements(5, 10).NilChance(0).Fuzz(&materials)
	}
	all = append(virtuals, materials...)
	pulse := gen.PulseNumber()

	t.Run("returns correct nodes", func(t *testing.T) {
		nodeStorage := NewStorage()
		nodeStorage.nodes[pulse] = all
		{
			result, err := nodeStorage.InRole(pulse, node.StaticRoleVirtual)
			assert.NoError(t, err)
			assert.Equal(t, virtuals, result)
		}
		{
			result, err := nodeStorage.InRole(pulse, node.StaticRoleLightMaterial)
			assert.NoError(t, err)
			assert.Equal(t, materials, result)
		}
	})

	t.Run("returns nil when empty nodes", func(t *testing.T) {
		nodeStorage := NewStorage()
		nodeStorage.nodes[pulse] = nil
		result, err := nodeStorage.InRole(pulse, node.StaticRoleVirtual)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error when no nodes", func(t *testing.T) {
		nodeStorage := NewStorage()
		result, err := nodeStorage.InRole(pulse, node.StaticRoleVirtual)
		assert.Equal(t, ErrNoNodes, err)
		assert.Nil(t, result)
	})
}

func TestStorage_Set(t *testing.T) {
	t.Parallel()

	var nodes []node.Node
	f := fuzz.New().Funcs(func(e *node.Node, c fuzz.Continue) {
		e.ID = gen.UniqueReference()
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
		err := nodeStorage.Set(pulse, []node.Node{})
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
