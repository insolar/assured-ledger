// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package filaments

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/dropbag"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
)

type WriteEntry struct {
	Key reference.Holder

	// local or remote
	// remote may have Prev = nil if prev is not read
	Prev   *WriteEntry
	Next   AtomicEntry
	Latest *LocalSegment // FilamentSegment

	// EntryHash == Key.Local

	EventSeq     uint32 // per JetDrop
	DirectorySeq uint32 // per JetDrop, index in Merkle tree
	FilamentSeq  uint64 // sequence through the whole filament, first entry =1, zero is invalid

	// StorageLocator uint64(?) // atomic

	BodyHash    cryptkit.Digest
	BodySection dropbag.JetSectionId
	Body        *WriteEntryBody

	// AuthRecordHash etc - an additional section for custom primary cryptography

	ProducerSignature  cryptkit.Signature // over BodyRecordHash + (?)AuthRecordHash
	RegistrarSignature cryptkit.Signature // over ProducerSignature + Sequences + Prev.Key.Local (EntryHash)
}

type WriteEntryBody struct {
	//	RecordHash  cryptkit.Digest

	ProducerNode  reference.Holder // TODO make specific type for Node ref
	RegistrarNode reference.Holder // TODO make specific type for Node ref

	LifelineRoot reference.Holder
	// OtherRef
	// ReasonRef

	// other sections
}

func (p *WriteEntry) LifelineRoot() reference.Holder {
	if ll := p.Latest.lifelineRoot; ll != nil {
		return ll
	}
	return p.FilamentRoot()
}

func (p *WriteEntry) FilamentRoot() reference.Holder {
	return p.Latest.filamentRoot
}

func (p *WriteEntry) FilamentSection() dropbag.JetSectionId {
	return p.Latest.filamentSection
}
