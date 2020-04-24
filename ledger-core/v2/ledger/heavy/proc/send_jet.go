// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package proc

import (
	"context"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/bus"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
)

type SendJet struct {
	meta payload.Meta

	dep struct {
		jets   jet.Accessor
		sender bus.Sender
	}
}

func (p *SendJet) Dep(
	jets jet.Accessor,
	sender bus.Sender,
) {
	p.dep.jets = jets
	p.dep.sender = sender
}

func NewSendJet(meta payload.Meta) *SendJet {
	return &SendJet{
		meta: meta,
	}
}

func (p *SendJet) Proceed(ctx context.Context) error {
	getJet := payload.GetJet{}
	err := getJet.Unmarshal(p.meta.Payload)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal GetJet message")
	}

	jetID, actual := p.dep.jets.ForID(ctx, getJet.PulseNumber, getJet.ObjectID)

	msg, err := payload.NewMessage(&payload.Jet{
		JetID:  jetID,
		Actual: actual,
	})
	if err != nil {
		return errors.Wrap(err, "failed to create reply")
	}

	p.dep.sender.Reply(ctx, p.meta, msg)
	return nil
}
