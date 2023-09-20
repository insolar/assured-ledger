package ledger

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type ControlLocator StorageLocator
const MaxControlChapterID = MaxChapterID>>1

func NewControlLocator(chapterID ChapterID, ofs uint32) ControlLocator {
	switch {
	case chapterID == 0:
		panic(throw.IllegalValue())
	case chapterID > MaxControlChapterID:
		panic(throw.IllegalValue())
	case ofs > MaxChapterOffset:
		panic(throw.IllegalValue())
	}
	return ControlLocator(chapterID) << 24 | ControlLocator(ofs)
}

func NewDropControlLocator(id jet.ExactID, chapterID ChapterID, ofs uint64) ControlLocator {
	switch {
	case chapterID == 0:
		panic(throw.IllegalValue())
	case chapterID > MaxChapterID:
		panic(throw.IllegalValue())
	case id == 0:
		panic(throw.IllegalValue())
	case ofs > MaxChapterOffset:
		panic(throw.IllegalValue())
	}
	return ControlLocator(id)<<47 | ControlLocator(chapterID) << 24 | ControlLocator(ofs)
}

func (v ControlLocator) IsZero() bool {
	return v == 0
}

func (v ControlLocator) ExactID() jet.ExactID {
	return jet.ExactID(v >> 47)
}

func (v ControlLocator) ChapterID() ChapterID {
	return ChapterID(v >> 24) & MaxChapterID
}

func (v ControlLocator) ChapterOffset() uint32 {
	return uint32(v) & MaxChapterOffset
}

func (v ControlLocator) Offset() uint64 {
	return uint64(v) & 0x7FFF_FFFF_FFFF
}

/************************************/

func NewDropOrdinal(id jet.ExactID, ordinal Ordinal) DropOrdinal {
	switch {
	case id.IsZero():
		panic(throw.IllegalValue())
	case ordinal == 0:
		panic(throw.IllegalValue())
	}

	return DropOrdinal(id)<<32 | DropOrdinal(ordinal)
}

func NewNoDropOrdinal(ordinal Ordinal) DropOrdinal {
	if ordinal == 0 {
		panic(throw.IllegalValue())
	}

	return DropOrdinal(ordinal)
}

type DropOrdinal uint64

func (v DropOrdinal) Ordinal() Ordinal {
	return Ordinal(v)
}

func (v DropOrdinal) ExactID() jet.ExactID {
	return jet.ExactID(v>>32)
}

func (v DropOrdinal) IsZero() bool {
	return v == 0
}

