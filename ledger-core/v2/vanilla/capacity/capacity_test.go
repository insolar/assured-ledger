// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package capacity

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultPercent(t *testing.T) {
	require.Equal(t, 20, LevelMinimal.DefaultPercent())

	require.Panics(t, func() { Level(LevelCount).DefaultPercent() })
}

func TestChooseInt(t *testing.T) {
	var options [LevelCount]int
	l := LevelMinimal
	options[l] = 5
	require.Equal(t, 5, l.ChooseInt(options))

	require.Panics(t, func() { Level(LevelCount).ChooseInt(options) })
}
