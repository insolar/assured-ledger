// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package ledger

type SectionID uint16

const (
	// ControlSection is to keep storage management information. Resides outside of drops. Its entries can only be in this section.
	ControlSection SectionID = iota

	// DefaultEntrySection is to store catalog entries of a drop.
	DefaultEntrySection

	// DefaultDustSection is to store data temporarily with no exact guarantees of retention time after finalization.
	DefaultDustSection
)

// RelativeEntry is used to reference an entry in the same bundle for filament binding logic on storage side.
// Used by LineStage and UpdateBundle logic, which are not applicable for ControlSection entries.
const RelativeEntry = ControlSection

const MaxSectionID = (^SectionID(0))>>1

// DefaultDataSection is to store data indefinitely (except for wiping out & evictions)
const DefaultDataSection = DefaultEntrySection
