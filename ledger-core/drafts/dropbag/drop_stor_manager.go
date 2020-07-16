// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package dropbag

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/keyset"
)

type DropStorageManager interface {
	// will actually open CompositeDropStorage that may have multiple DropStorage(s) incorporated
	OpenStorage(jetId jet.ExactID, pn pulse.Number) (DropStorage, error)
	OpenPulseStorage(pn pulse.Number) (CompositeDropPerPulseData, error)

	// each storage is lazy-closed, but can be explicitly marked for closing
	UnusedStorage(CompositeDropStorage)

	// returns nil when the required storage is not open
	GetOpenedStorage(jetId jet.ExactID, pn pulse.Number) DropStorage

	BuildStorage(pr pulse.Range) CompositeDropStorageBuilder
}

type CloseRetainer interface {
	// guarantees that this object will not be closed until the returned Closer.Close() is called
	// multiple retainers are allowed, all of them must be closed to release the object
	// can return nil when retention guarantee is not possible (object is closed already)
	Retain() io.Closer
}

type CompositeDropStorageBuilder interface {
	CompositeDropStorageBuilder()
}

type CompositeDropPerPulseData interface {
	CoveringRange() pulse.Range // latest PulseData + earliest pulse number - will not include intermediate PulseData
	PulseDataCount() int
	GetPerPulseData(index int) DropPerPulseData
	FindPerPulseData(pulse.Number) DropPerPulseData
}

type CompositeDropStorage interface {
	CloseRetainer

	CoveringRange() pulse.Range // latest PulseData + earliest pulse number - will not include intermediate PulseData
	PerPulseData() CompositeDropPerPulseData

	// identified by the latest pulse in a range of the drop
	GetDropStorage(jetId jet.ExactID, pn pulse.Number) DropStorage

	// Jets
	// Cabinet -> StorageCabinet

	NoticeUsage()
}

type DropType uint8

const (
	_ DropType = iota
	RegularDropType
	SummaryDropType
	ArchivedDropType
)

type DropPerPulseData interface {
	PulseRange() pulse.Range // complete range, with all pulse data included
	JetTree() DropJetTree

	OnlinePopulation() census.OnlinePopulation
	OfflinePopulation() census.OfflinePopulation
}

type DropJetTree interface {
	MinDepth() uint8
	MaxDepth() uint8
	Count() int
	PrefixToJetId(prefix jet.Prefix) jet.ExactID
	KeyToJetId(keyset.Key) jet.ExactID
}

type DropStorage interface {
	CloseRetainer

	Composite() CompositeDropStorage

	JetId() jet.LegID
	PulseNumber() pulse.Number // the rightmost/latest pulse number of this drop
	PerPulseData() DropPerPulseData

	DropType() DropType

	// a synthetic directory based on a few sections, marked as primary
	MainDirectory() DropSectionDirectory

	FindSection(DropSectionId) DropSection
	FindDirectory(DropSectionId) DropSectionDirectory
}

type DropSectionId uint8

const (
	// a special section that can't be used directly
	DropControlSection DropSectionId = iota

	// main persistent section
	MainDropSection

	// limited persistence section
	DustDropSection
)

const MinCustomDropSection DropSectionId = 16

type DropSectionDirectory interface {
	DropSectionId() DropSectionId
	//IsPrimary()

	LookupKey(keyset.Key) (DropEntry, bool)
	LookupKeySet(keyset.KeySet) LookupPager
}

type LookupMiss uint8

const (
	LookupUnknown LookupMiss = iota
	LookupNotFound
)

type LookupPageFunc func(estTotal, curTotal uint, found []DropEntry, misses [] /* LookupMiss */ []keyset.Key, skipped uint) bool

type LookupPager interface {
	RequestedKeySet() keyset.KeySet

	LoadKeys(maxPageSize uint, fn LookupPageFunc) error
	PeekKeys(maxPageSize uint, fn LookupPageFunc)
}

type DropEntry interface {
	Key() keyset.Key

	// actual directory this entry is listed in
	DirectorySectionId() DropSectionId
	// a sequential order for all entries being of a drop
	// for regular entry starts from 256 and goes up
	SequenceId() uint32

	// section of entry's content
	ContentSectionId() DropSectionId
	//	ContentStorageLocator() // StorageLocator

	PeekEntry(ObjectExtractor) bool
	GetEntryUnbound(ObjectExtractor) (value interface{}, hasValue bool)
}

type DropSection interface {
	DropSectionId() DropSectionId
	DirectorySectionId() DropSectionId

	// CryptographyProvider() SectionCryptographyProvider
	// CryptographyPolicy() SectionCryptographyPolicy
	// RetentionPolicy() SectionRetentionPolicy

	LoadEntry(DropEntry, ObjectExtractor) error
	LoadEntries([]DropEntry, ObjectExtractor) error
}

type ExtractorId string

// This interface is a combination of data extractor and consumer.
// It extracts (converts) from raw data into a target type, then process it.
type ObjectExtractor interface {
	// extractors with the same id will share cached values etc
	ExtractorId() ExtractorId

	// Invoked when an entry has no cached value for this extractor id.
	// Extracts required data from the raw bytes and consumes it (process further).
	// The extracted result must be returned for caching.
	// Return:
	// 	(cacheValue) the extracted value
	// 	(boundValue) true when cacheValue depends on (raw), as it can be memory mapped and needs cleanup
	//  (estSize) approximate memory size of the extracted value, for cache eviction procedure.
	Extract(key keyset.Key, raw []byte) (cacheValue interface{}, boundValue bool, estSize uint)

	// Invoked when an entry has cached value for this extractor id.
	Reuse(key keyset.Key, cacheValue interface{}, boundValue bool)

	// Invoked by GetUnboundObject() when a cached value is bound.
	// This function must return an unbound copy of the (cacheValue).
	Unbind(cacheValue interface{}) interface{}
}
