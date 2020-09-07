// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package ledger

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type ChapterID uint32
const MaxChapterID ChapterID = 0x00FF_FFFF
const MaxChapterOffset = 0x00FF_FFFF

type StorageLocator uint64

func NewLocator(id SectionID, chapterID ChapterID, ofs uint32) StorageLocator {
	switch {
	case chapterID == 0:
		panic(throw.IllegalValue())
	case chapterID > MaxChapterID:
		panic(throw.IllegalValue())
	case ofs > MaxChapterOffset:
		panic(throw.IllegalValue())
	}
	return StorageLocator(id)<<48 | StorageLocator(chapterID) << 24 | StorageLocator(ofs)
}

func NewOffsetLocator(id SectionID, ofs uint64) StorageLocator {
	if ofs > 0xFFFF_FFFF_FFFF {
		panic(throw.IllegalValue())
	}
	return StorageLocator(id)<<48 | StorageLocator(ofs)
}

func (v StorageLocator) IsZero() bool {
	return v == 0
}

func (v StorageLocator) SectionID() SectionID {
	return SectionID(v >> 48)
}

func (v StorageLocator) ChapterID() ChapterID {
	return ChapterID(v >> 24) & MaxChapterID
}

func (v StorageLocator) ChapterOffset() uint32 {
	return uint32(v) & MaxChapterOffset
}

func (v StorageLocator) Offset() uint64 {
	return uint64(v) & 0xFFFF_FFFF_FFFF
}

