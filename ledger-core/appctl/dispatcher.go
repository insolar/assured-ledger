// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package appctl

import (
	"time"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type Message = message.Message

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/appserver.Dispatcher -o ./ -s _mock.go -g
type Dispatcher interface {
	PreparePulseChange(PulseChange, NodeStateSink)
	CancelPulseChange()
	CommitPulseChange(PulseChange)
	Process(msg *Message) error
}

type PulseChange struct {
	PulseSeq  uint32
	Pulse     pulse.Range
	StartedAt time.Time
	Census    census.Operational
}

type MessageTag struct {
	PulseNum  pulse.Number
	Source    reference.Holder
}
