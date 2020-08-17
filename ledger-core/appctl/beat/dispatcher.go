// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package beat

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/appctl/beat.Dispatcher -o ./ -s _mock.go -g
type Dispatcher interface {
	PrepareBeat(Ack)
	CancelBeat()
	CommitBeat(Beat)
	Process(Message) error
}
