// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package executor

import (
	"context"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/bus"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/record"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/ledger/light/executor.DetachedNotifier -o ./ -s _mock.go -g

type DetachedNotifier interface {
	Notify(
		ctx context.Context,
		openedRequests []record.CompositeFilamentRecord,
		objectID insolar.ID,
		closedRequestID insolar.ID,
	)
}

type DetachedNotifierDefault struct {
	sender bus.Sender
}

func NewDetachedNotifierDefault(
	sender bus.Sender,
) *DetachedNotifierDefault {
	return &DetachedNotifierDefault{
		sender: sender,
	}
}

// Notify sends notifications about detached requests that are ready for execution.
func (p *DetachedNotifierDefault) Notify(
	ctx context.Context,
	openedRequests []record.CompositeFilamentRecord,
	objectID insolar.ID,
	closedRequestID insolar.ID,
) {
	for _, req := range openedRequests {
		outgoing, ok := record.Unwrap(&req.Record.Virtual).(*record.OutgoingRequest)
		if !ok {
			continue
		}
		if !outgoing.IsDetachedCall() {
			continue
		}
		if reasonRef := outgoing.ReasonRef(); *reasonRef.GetLocal() != closedRequestID {
			continue
		}

		buf, err := req.Record.Virtual.Marshal()
		if err != nil {
			inslogger.FromContext(ctx).Error(
				errors.Wrapf(err, "failed to notify about detached %s", req.RecordID.DebugString()),
			)
			continue
		}
		msg, err := payload.NewMessage(&payload.SagaCallAcceptNotification{
			ObjectID:          objectID,
			DetachedRequestID: req.RecordID,
			Request:           buf,
		})
		if err != nil {
			inslogger.FromContext(ctx).Error(
				errors.Wrapf(err, "failed to notify about detached %s", req.RecordID.DebugString()),
			)
			continue
		}
		_, done := p.sender.SendRole(ctx, msg, insolar.DynamicRoleVirtualExecutor, *insolar.NewReference(objectID))
		done()
	}
}
