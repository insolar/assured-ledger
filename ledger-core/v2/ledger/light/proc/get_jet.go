// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package proc

import (
	"context"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/bus"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
)

type GetJet struct {
	message  payload.Meta
	objectID insolar.ID
	pulse    insolar.PulseNumber

	dep struct {
		jets   jet.Accessor
		sender bus.Sender
	}
}

func (p *GetJet) Dep(
	jets jet.Accessor,
	sender bus.Sender,
) {
	p.dep.jets = jets
	p.dep.sender = sender
}

func NewGetJet(msg payload.Meta, objectID insolar.ID, pulse insolar.PulseNumber) *GetJet {
	return &GetJet{
		message:  msg,
		objectID: objectID,
		pulse:    pulse,
	}
}

func (p *GetJet) Proceed(ctx context.Context) error {
	jetID, actual := p.dep.jets.ForID(ctx, p.pulse, p.objectID)

	msg, err := payload.NewMessage(&payload.Jet{
		JetID:  jetID,
		Actual: actual,
	})
	if err != nil {
		return errors.Wrap(err, "GetJet: failed to create reply")
	}

	p.dep.sender.Reply(ctx, p.message, msg)
	return nil
}
