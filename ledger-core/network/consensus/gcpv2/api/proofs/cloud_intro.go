package proofs

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

type NodeWelcomePackage struct {
	CloudIdentity      cryptkit.DigestHolder
	LastCloudStateHash cryptkit.DigestHolder
	JoinerSecret       cryptkit.DigestHolder // can be nil
}
