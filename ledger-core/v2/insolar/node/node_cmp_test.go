// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package node_test

import (
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
)

func TestNode(t *testing.T) {
	var (
		virtuals  []insolar.Node
		materials []insolar.Node
		all       []insolar.Node
	)
	{
		f := fuzz.New().Funcs(func(e *insolar.Node, c fuzz.Continue) {
			e.ID = gen.Reference()
			e.Role = insolar.StaticRoleVirtual
		})
		f.NumElements(5, 10).NilChance(0).Fuzz(&virtuals)
	}
	{
		f := fuzz.New().Funcs(func(e *insolar.Node, c fuzz.Continue) {
			e.ID = gen.Reference()
			e.Role = insolar.StaticRoleLightMaterial
		})
		f.NumElements(5, 10).NilChance(0).Fuzz(&materials)
	}
	all = append(virtuals, materials...)
	pulse := gen.PulseNumber()
	storage := node.NewStorage()

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
		result, err := storage.InRole(pulse, insolar.StaticRoleVirtual)
		assert.NoError(t, err)
		assert.Equal(t, virtuals, result)
	}
	// Deletes nodes.
	{
		storage.DeleteForPN(pulse)
		result, err := storage.All(pulse)
		assert.Equal(t, node.ErrNoNodes, err)
		assert.Nil(t, result)
	}
}
