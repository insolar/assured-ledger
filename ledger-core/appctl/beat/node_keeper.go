// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package beat

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/appctl/beat.NodeKeeper -s _mock.go -g

type NodeKeeper interface {
	NodeNetwork

	AddExpectedBeat(Beat) error
	AddCommittedBeat(Beat) error
}

