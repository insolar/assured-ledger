// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nodestorage_test

import (
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/nodestorage"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
)

func TestNode(t *testing.T) {
	var (
		virtuals  []node.Node
		materials []node.Node
		all       []node.Node
	)
	{
		f := fuzz.New().Funcs(func(e *node.Node, c fuzz.Continue) {
			e.ID = gen.UniqueGlobalRef()
			e.Role = node.StaticRoleVirtual
		})
		f.NumElements(5, 10).NilChance(0).Fuzz(&virtuals)
	}
	{
		f := fuzz.New().Funcs(func(e *node.Node, c fuzz.Continue) {
			e.ID = gen.UniqueGlobalRef()
			e.Role = node.StaticRoleLightMaterial
		})
		f.NumElements(5, 10).NilChance(0).Fuzz(&materials)
	}
	all = append(virtuals, materials...)
	pulse := gen.PulseNumber()
	storage := nodestorage.NewStorage()

	// Saves nodes.
	{
		err := storage.Set(pulse, all)
		assert.NoError(t, err)
	}
	// Returns all nodes.
	{
		result, err := storage.All(pulse)
		assert.NoError(t, err)
		assert.Equal(t, all, result)
	}
	// Returns in role nodes.
	{
		result, err := storage.InRole(pulse, node.StaticRoleVirtual)
		assert.NoError(t, err)
		assert.Equal(t, virtuals, result)
	}
	// Deletes nodes.
	{
		storage.DeleteForPN(pulse)
		result, err := storage.All(pulse)
		assert.Equal(t, nodestorage.ErrNoNodes, err)
		assert.Nil(t, result)
	}
}
