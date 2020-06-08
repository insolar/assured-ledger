// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package sequence

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGenerator(t *testing.T) {
	gen := NewGenerator()

	require.Equal(t, gen, &generator{sequence: new(uint64)})
}

func TestGenerator_Generate(t *testing.T) {
	gen := NewGenerator()

	seq1 := gen.Generate()
	assert.Equal(t, seq1, Sequence(1))

	seq2 := gen.Generate()
	assert.Equal(t, seq2, Sequence(2))

	seq3 := gen.Generate()
	assert.Equal(t, seq3, Sequence(3))
}
