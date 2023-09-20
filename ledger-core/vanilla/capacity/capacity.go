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

func (v Level) DefaultPercent() int {
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

func (v Level) ChooseFloat32(levels Float32Levels) float32 {
	return levels[v]
}
