// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package drop

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/ledger/drop.Modifier -o ./ -s _mock.go -g

// Modifier provides an interface for modifying jetdrops.
type Modifier interface {
	Set(ctx context.Context, drop Drop) error
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/ledger/drop.Accessor -o ./ -s _mock.go -g

// Accessor provides an interface for accessing jetdrops.
type Accessor interface {
	ForPulse(ctx context.Context, jetID insolar.JetID, pulse insolar.PulseNumber) (Drop, error)
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/ledger/drop.Cleaner -o ./ -s _mock.go -g

// Cleaner provides an interface for removing jetdrops from a storage.
type Cleaner interface {
	DeleteForPN(ctx context.Context, pulse insolar.PulseNumber)
}
