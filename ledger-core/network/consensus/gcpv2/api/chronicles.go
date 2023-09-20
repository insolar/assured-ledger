package api

import (
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

type ConsensusChronicles interface {
	GetProfileFactory(ksf cryptkit.KeyStoreFactory) profiles.Factory

	GetActiveCensus() census.Active
	GetExpectedCensus() census.Expected
	GetLatestCensus() (lastCensus census.Operational, expectedCensus bool)
	GetRecentCensus(pn pulse.Number) census.Operational
	// FindArchivedCensus(pn common.Number) Archived
}
