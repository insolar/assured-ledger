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

func TestChooseUint(t *testing.T) {
	require.Equal(t, uint(5), LevelMinimal.ChooseUint([LevelCount]uint{LevelMinimal: 5}))
}

func TestChooseUint32(t *testing.T) {
	require.Equal(t, uint32(5), LevelMinimal.ChooseUint32([LevelCount]uint32{LevelMinimal: 5}))
}

func TestChooseUint8(t *testing.T) {
	require.Equal(t, uint8(5), LevelMinimal.ChooseUint8([LevelCount]uint8{LevelMinimal: 5}))
}

func TestChooseFloat32(t *testing.T) {
	require.Equal(t, float32(5.5), LevelMinimal.ChooseFloat32([LevelCount]float32{LevelMinimal: 5.5}))
}
