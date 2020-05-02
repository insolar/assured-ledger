// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package jet

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/insolar/jet.Coordinator -o ./ -s _mock.go -g

// Coordinator provides methods for calculating Jet affinity
// (e.g. to which Jet a message should be sent).
type Coordinator interface {
	// Me returns current node.
	Me() insolar.Reference

	// QueryRole returns node refs responsible for role bound operations for given object and pulse.
	QueryRole(ctx context.Context, role insolar.DynamicRole, obj insolar.ID, pulse insolar.PulseNumber) ([]insolar.Reference, error)
}
