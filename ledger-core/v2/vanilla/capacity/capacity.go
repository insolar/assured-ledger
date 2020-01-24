//
//    Copyright 2019 Insolar Technologies
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.
//

package capacity

type Level uint8

const (
	LevelZero Level = iota
	LevelMinimal
	LevelReduced
	LevelNormal
	LevelMax
)

const LevelCount = LevelMax + 1

func (v Level) DefaultPercent() int {
	// 0, 25, 75, 100, 125
	return v.ChooseInt([...]int{0, 20, 60, 80, 100})
}

func (v Level) ChooseInt(options [LevelCount]int) int {
	return options[v]
}
