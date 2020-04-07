// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package capacity

type Level uint8

const (
	LevelZero Level = iota
	LevelMinimal
	LevelReduced
	LevelNormal
	LevelMax

	LevelCount = iota
)

type IntLevels [LevelCount]int
type UintLevels [LevelCount]uint
type Uint8Levels [LevelCount]uint8
type Uint32Levels [LevelCount]uint32
type Float32Levels [LevelCount]float32

//const LevelCount = LevelMax + 1

func (v Level) DefaultPercent() int {
	// 0, 25, 75, 100, 125
	return v.ChooseInt([...]int{0, 20, 60, 80, 100})
}

func (v Level) ChooseInt(levels IntLevels) int {
	return levels[v]
}

func (v Level) ChooseUint(levels UintLevels) uint {
	return levels[v]
}

func (v Level) ChooseUint32(levels Uint32Levels) uint32 {
	return levels[v]
}

func (v Level) ChooseUint8(levels Uint8Levels) uint8 {
	return levels[v]
}

//func (v Level) ChooseByteOf(levels uint32) uint8 {
//	return levels[v]
//}

func (v Level) ChooseFloat32(levels Float32Levels) float32 {
	return levels[v]
}
