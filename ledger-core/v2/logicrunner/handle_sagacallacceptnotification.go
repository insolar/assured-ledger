package logicrunner

import (
	"context"
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/flow"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/record"
)

type HandleSagaCallAcceptNotification struct {
	dep  *Dependencies
	meta payload.Meta
}

func (h *HandleSagaCallAcceptNotification) Present(ctx context.Context, f flow.Flow) error {
	msg := payload.SagaCallAcceptNotification{}
	err := msg.Unmarshal(h.meta.Payload)
	if err != nil {
		return err
	}

	virtual := record.Virtual{}
	err = virtual.Unmarshal(msg.Request)
	if err != nil {
		return err
	}
	rec := record.Unwrap(&virtual)
	outgoing, ok := rec.(*record.OutgoingRequest)
	if !ok {
		return fmt.Errorf("unexpected request received %T", rec)
	}

	if err := checkOutgoingRequest(ctx, outgoing); err != nil {
		return err
	}

	outgoingReqRef := insolar.NewRecordReference(msg.DetachedRequestID)
	_, _, err = h.dep.OutgoingSender.SendOutgoingRequest(ctx, *outgoingReqRef, outgoing)
	return err
}
