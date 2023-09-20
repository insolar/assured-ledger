package dropbag

import (
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

type JetPulse interface {
	GetOnlinePopulation() census.OnlinePopulation
	GetPulseData() pulse.Range
}

type JetSectionId uint16

const (
	DefaultSection JetSectionId = iota // MapSection
	ControlSection                     // drop/dropbag lifecycle
	DustSection                        // transient, general, stays for some time (e.g. log)
	GasSection                         // transient, requests, stays until processed
)

type JetDrop interface {
	PulseNumber() pulse.Number
	GetGlobulaPulse() JetPulse

	FindEntryByKey(longbits.ByteString) JetDropEntry
	//	GetSectionDirectory(JetSectionId) JetDropSection
	GetSection(JetSectionId) JetDropSection
}

type JetSectionType uint8

const (
	DirectorySection JetSectionType = 1 << iota //
	TransientSection
	CustomCryptographySection
	//HeavyPayload
)

type JetSectionDeclaration interface {
	HasDirectory() bool
	//IsSorted
	HasPayload() bool
}

type JetSectionDirectory interface {
	FindByKey(longbits.ByteString) JetDropEntry
	EnumKeys()
}

type JetDropSection interface {
	EnumEntries()
}

type JetDropEntry interface {
	Key() longbits.ByteString
	Section() JetDropSection
	IsAvailable() bool
	Data() []byte
	// ProjectionCache()
}

type KeySet interface {
	// inclusive or exclusive

}
