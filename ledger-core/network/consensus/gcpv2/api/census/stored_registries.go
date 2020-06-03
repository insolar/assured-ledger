// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package census

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/common/endpoints"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/gcpv2/api/misbehavior"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/gcpv2/api/proofs"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/gcpv2/api/census.VersionedRegistries -o . -s _mock.go -g

type VersionedRegistries interface {
	// GetVersionId() int
	CommitNextPulse(pd pulse.Data, population OnlinePopulation) VersionedRegistries

	GetMisbehaviorRegistry() MisbehaviorRegistry
	GetMandateRegistry() MandateRegistry
	GetOfflinePopulation() OfflinePopulation
	GetVersionPulseData() pulse.Data
	GetNearestValidPulseData() pulse.Data
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/gcpv2/api/census.MisbehaviorRegistry -o . -s _mock.go -g

type MisbehaviorRegistry interface {
	AddReport(report misbehavior.Report)
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/gcpv2/api/census.MandateRegistry -o . -s _mock.go -g

type MandateRegistry interface {
	FindRegisteredProfile(host endpoints.Inbound) profiles.Host
	GetPrimingCloudHash() proofs.CloudStateHash
	GetCloudIdentity() cryptkit.DigestHolder
	GetConsensusConfiguration() ConsensusConfiguration
}

type ConsensusConfiguration interface {
}
