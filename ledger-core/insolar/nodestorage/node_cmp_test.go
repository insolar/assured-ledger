// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nodestorage_test

import (
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/insolar/nodestorage"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
)

func TestNode(t *testing.T) {
	var (
		virtuals  []rms.Node
		materials []rms.Node
		all       []rms.Node
	)
	{
		f := fuzz.New().Funcs(func(e *rms.Node, c fuzz.Continue) {
			e.ID.Set(gen.UniqueGlobalRef())
			e.Role = member.PrimaryRoleVirtual
		})
		f.NumElements(5, 10).NilChance(0).Fuzz(&virtuals)
	}
	{
		f := fuzz.New().Funcs(func(e *rms.Node, c fuzz.Continue) {
			e.ID.Set(gen.UniqueGlobalRef())
			e.Role = member.PrimaryRoleLightMaterial
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
		result, err := storage.InRole(pulse, member.PrimaryRoleVirtual)
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
