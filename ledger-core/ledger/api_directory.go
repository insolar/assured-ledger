package ledger

import (
	"math"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type Ordinal uint32
type ExtensionID uint32

const (
	OrdinalSize = 4
	SameAsBodyExtensionID ExtensionID = 0
)

type DirectoryIndex uint64

func NewDirectoryIndex(id SectionID, ordinal Ordinal) DirectoryIndex {
	switch {
	case id > MaxSectionID:
		panic(throw.IllegalValue())
	case ordinal == 0:
		panic(throw.IllegalValue())
	}
	return DirectoryIndex(id)<<48 | DirectoryIndex(ordinal)
}

func (v DirectoryIndex) IsZero() bool {
	return v == 0
}

func (v DirectoryIndex) SectionID() SectionID {
	return SectionID(v >> 48)
}

func (v DirectoryIndex) Ordinal() Ordinal {
	return Ordinal(v)
}

func (v DirectoryIndex) WithFlags(flags DirectoryEntryFlags) DirectoryIndexAndFlags {
	return DirectoryIndexAndFlags(v) | DirectoryIndexAndFlags(flags)<<32
}

type DirectoryIndexAndFlags uint64

func (v DirectoryIndexAndFlags) IsZero() bool {
	return v == 0
}

func (v DirectoryIndexAndFlags) SectionID() SectionID {
	return SectionID(v >> 48)
}

func (v DirectoryIndexAndFlags) Ordinal() Ordinal {
	return Ordinal(v)
}

func (v DirectoryIndexAndFlags) DirectoryIndex() DirectoryIndex {
	return DirectoryIndex(v) &^ (math.MaxUint16<<32)
}

func (v DirectoryIndexAndFlags) Flags() DirectoryEntryFlags {
	return DirectoryEntryFlags(v >> 32)
}

func (v DirectoryIndexAndFlags) WithIndex(index DirectoryIndex) DirectoryIndexAndFlags {
	return (v & (math.MaxUint16<<32)) | DirectoryIndexAndFlags(index)
}

func (v DirectoryIndexAndFlags) WithFlags(flags DirectoryEntryFlags) DirectoryIndexAndFlags {
	return (v &^ (math.MaxUint16<<32)) | DirectoryIndexAndFlags(flags)<<32
}

type DirectoryEntryFlags uint16

const (
	FilamentClose DirectoryEntryFlags = 1<<iota
	FilamentLocalStart
	FilamentReopen
)
