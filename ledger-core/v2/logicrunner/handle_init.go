// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package logicrunner

import (
	"context"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/record"
)

func CheckOutgoingRequest(ctx context.Context, request *record.OutgoingRequest) error {
	return CheckIncomingRequest(ctx, (*record.IncomingRequest)(request))
}

func CheckIncomingRequest(_ context.Context, request *record.IncomingRequest) error {
	if !request.CallerPrototype.IsEmpty() && !request.CallerPrototype.IsObjectReference() {
		return errors.Errorf("request.CallerPrototype should be ObjectReference; ref=%s", request.CallerPrototype.String())
	}
	if request.Base != nil && !request.Base.IsObjectReference() {
		return errors.Errorf("request.Base should be ObjectReference; ref=%s", request.Base.String())
	}
	if request.Object != nil && !request.Object.IsObjectReference() {
		return errors.Errorf("request.Object should be ObjectReference; ref=%s", request.Object.String())
	}
	if request.Prototype != nil && !request.Prototype.IsObjectReference() {
		return errors.Errorf("request.Prototype should be ObjectReference; ref=%s", request.Prototype.String())
	}
	if request.Reason.IsEmpty() || !request.Reason.IsRecordScope() {
		return errors.Errorf("request.Reason should be RecordReference; ref=%s", request.Reason.String())
	}

	if rEmpty, cEmpty := request.APINode.IsEmpty(), request.Caller.IsEmpty(); rEmpty == cEmpty {
		rStr := "Caller is empty"
		if !rEmpty {
			rStr = "Caller is not empty"
		}
		cStr := "APINode is empty"
		if !cEmpty {
			cStr = "APINode is not empty"
		}

		return errors.Errorf("failed to check request origin: one should be set, but %s and %s", rStr, cStr)
	}

	if !request.Caller.IsEmpty() && !request.Caller.IsObjectReference() {
		return errors.Errorf("request.Caller should be ObjectReference; ref=%s", request.Caller.String())
	}
	if !request.APINode.IsEmpty() && !request.APINode.IsObjectReference() {
		return errors.Errorf("request.APINode should be ObjectReference; ref=%s", request.APINode.String())
	}

	return nil
}
