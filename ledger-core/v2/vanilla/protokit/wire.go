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

package protokit

import (
	"fmt"
	"io"
	"math"
)

type WireTag uint32

func SafeWireTag(v uint64) (WireTag, error) {
	if v > math.MaxUint32 {
		return 0, fmt.Errorf("invalid wire tag, overflow, %x", v)
	}
	wt := WireTag(v)
	if wt.IsValid() {
		return wt, nil
	}
	return 0, fmt.Errorf("invalid wire tag: %v", v)
}

func (v WireTag) IsZero() bool {
	return v != 0
}

func (v WireTag) IsValid() bool {
	return v.FieldId() > 0 && v.Type().IsValid()
}

func (v WireTag) Type() WireType {
	return WireType(v & maskWireType)
}

func (v WireTag) FieldId() int {
	return int(v >> WireTypeBits)
}

func (v WireTag) TagSize() int {
	return SizeVarint32(uint32(v))
}

func (v WireTag) MaxFieldSize() (isFixed bool, max int) {
	switch minSize, maxSize := v.Type().DataSize(); {
	case minSize == maxSize:
		return true, maxSize + v.TagSize()
	case maxSize > 0:
		maxSize += v.TagSize()
		return false, maxSize
	default:
		return false, int(math.MaxInt64)
	}
}

func (v WireTag) FieldSize(u uint64) uint64 {
	return v.Type().FieldSize(v.TagSize(), u)
}

func (v WireTag) FixedFieldSize() int {
	if isFixed, maxSize := v.MaxFieldSize(); !isFixed {
		panic("illegal state - not fixed size")
	} else {
		return maxSize
	}
}

func (v WireTag) EnsureFixedFieldSize(sz int) WireTag {
	if v.FixedFieldSize() != sz {
		panic("illegal value - size mismatched")
	}
	return v
}

func (v WireTag) _checkTag(expected WireTag) error {
	if v == expected {
		return nil
	}
	return fmt.Errorf("tag mismatch: actual=%v, expected=%v", expected, v)
}

func (v WireTag) CheckType(t WireType) error {
	switch {
	case !t.IsValid():
		panic("illegal value")
	case t != v.Type():
		return fmt.Errorf("type mismatch: actual=%v, expectedType=%v", v, t)
	}
	return nil
}

func (v WireTag) CheckTag(expected WireTag) error {
	if !expected.IsValid() {
		panic("illegal value")
	}
	return v._checkTag(expected)
}

func (v WireTag) Check(expectedType WireType, expectedId int) error {
	return v._checkTag(expectedType.Tag(expectedId))
}

func (v WireTag) ReadValue(r io.ByteReader) (uint64, error) {
	return v.Type().ReadValue(r)
}

func (v WireTag) WriteValue(w io.ByteWriter, u uint64) error {
	return v.Type().WriteValue(w, u)
}

func (v WireTag) WriteTagValue(w io.ByteWriter, u uint64) error {
	if err := EncodeVarint(w, uint64(v)); err != nil {
		return err
	}
	return v.Type().WriteValue(w, u)
}

func (v WireTag) EnsureType(expectedType WireType) {
	if err := v.CheckType(expectedType); err != nil {
		panic(err)
	}
}

func (v WireTag) EnsureTag(expected WireTag) {
	if err := v.CheckTag(expected); err != nil {
		panic(err)
	}
}

func (v WireTag) Ensure(expectedType WireType, expectedId int) {
	if err := v.Check(expectedType, expectedId); err != nil {
		panic(err)
	}
}

func (v WireTag) MustWrite(w io.ByteWriter, u uint64) {
	if err := v.WriteTagValue(w, u); err != nil {
		panic(err)
	}
}

func (v WireTag) MustRead(r io.ByteReader) uint64 {
	if u, err := v.ReadTagValue(r); err != nil {
		panic(err)
	} else {
		return u
	}
}

func (v WireTag) String() string {
	if v == 0 {
		return "zeroTag"
	}
	return fmt.Sprintf("%d:%v", v.FieldId(), v.Type())
}
