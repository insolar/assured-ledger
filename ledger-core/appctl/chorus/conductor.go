// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package chorus

import (
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/appctl/chorus.Conductor -s _mock.go -g
// Conductor provides methods to orchestrate pulses through application's components.
type Conductor interface {
	// Set set's new pulse and closes current jet drop.
	CommitPulseChange(beat.Beat) error
}

