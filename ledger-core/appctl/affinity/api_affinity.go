package affinity

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/appctl/affinity.Helper -o ./ -s _mock.go -g

// Helper provides methods for calculating Jet affinity
// (e.g. to which Jet a message should be sent).
type Helper interface {
	// Me returns current node.
	Me() reference.Global

	// QueryRole returns node refs responsible for role bound operations for given object and pulse.
	QueryRole(role DynamicRole, obj reference.Holder, pulse pulse.Number) ([]reference.Global, error)
}
