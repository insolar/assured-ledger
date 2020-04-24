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
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/flow"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger/object"
)

type HasPendings struct {
	message  payload.Meta
	objectID insolar.ID

	dep struct {
		index  object.MemoryIndexAccessor
		sender bus.Sender
	}
}

func NewHasPendings(msg payload.Meta, objectID insolar.ID) *HasPendings {
	return &HasPendings{
		message:  msg,
		objectID: objectID,
	}
}

func (hp *HasPendings) Dep(
	index object.MemoryIndexAccessor,
	sender bus.Sender,
) {
	hp.dep.index = index
	hp.dep.sender = sender
}

func (hp *HasPendings) Proceed(ctx context.Context) error {
	idx, err := hp.dep.index.ForID(ctx, flow.Pulse(ctx), hp.objectID)
	if err != nil {
		return err
	}

	msg, err := payload.NewMessage(&payload.PendingsInfo{
		HasPendings: idx.Lifeline.EarliestOpenRequest != nil && *idx.Lifeline.EarliestOpenRequest < flow.Pulse(ctx),
	})
	if err != nil {
		return errors.Wrap(err, "failed to create reply")
	}

	hp.dep.sender.Reply(ctx, hp.message, msg)
	return nil
}
