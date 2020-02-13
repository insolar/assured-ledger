// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insolar

import (
	"context"
)

// APIRunner
type APIRunner interface {
	IsAPIRunner() bool
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/insolar.AvailabilityChecker -o ../testutils -s _mock.go -g

// AvailabilityChecker component checks if insolar network can't process any new requests
type AvailabilityChecker interface {
	IsAvailable(context.Context) bool
}
