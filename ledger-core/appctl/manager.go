// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package appctl

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/appctl.Manager -s _mock.go -g
// Manager provides Ledger's methods related to Pulse.
type Manager interface {
	// Set set's new pulse and closes current jet drop. If dry is true, nothing will be saved to storage.
	CommitPulseChange(PulseChange) error
}

