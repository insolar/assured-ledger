// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package jet

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/insolar/jet.AffinityHelper -o ./ -s _mock.go -g

// AffinityHelper provides methods for calculating Jet affinity
// (e.g. to which Jet a message should be sent).
type AffinityHelper interface {
	// Me returns current node.
	Me() reference.Global

	// QueryRole returns node refs responsible for role bound operations for given object and pulse.
	QueryRole(ctx context.Context, role node.DynamicRole, obj reference.Holder, pulse pulse.Number) ([]reference.Global, error)
}
