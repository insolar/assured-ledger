// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package catalog

type SectionID uint16

const (
	// ControlSection is to keep storage management information. Resides outside of drops. Its entries can only be in this section.
	ControlSection SectionID = iota

	// DefaultEntrySection is to store catalog entries of a drop.
	DefaultEntrySection

	// DefaultDataSection is to store data indefinitely (except for wiping out & evictions)
	DefaultDataSection

	// DefaultDustSection is to store data temporarily with no exact guarantees of retention time after finalization.
	DefaultDustSection
)

type DirectoryIndex uint64

func (v DirectoryIndex) IsZero() bool {
	return v == 0
}

func (v DirectoryIndex) SectionID() SectionID {
	return SectionID(v >> 48)
}

func (v DirectoryIndex) Index() uint32 {
	return uint32(v)
}

type ChapterID uint32
type StorageLocator uint64

func (v StorageLocator) IsZero() bool {
	return v == 0
}

func (v StorageLocator) SectionID() SectionID {
	return SectionID(v >> 48)
}

func (v StorageLocator) ChapterID() ChapterID {
	return ChapterID(v >> 24) & 0x00FF_FFFF
}

func (v StorageLocator) ChapterOffset() uint32 {
	return uint32(v) & 0x00FF_FFFF
}

func (v StorageLocator) Offset() uint64 {
	return uint64(v) & 0xFFFF_FFFF_FFFF
}

