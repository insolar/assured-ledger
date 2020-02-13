// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package outgoing

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/record"
)

func BuildIncomingRequestFromOutgoing(outgoing *record.OutgoingRequest) *record.IncomingRequest {
	// Currently IncomingRequest and OutgoingRequest are almost exact copies of each other
	// thus the following code is a bit ugly. However this will change when we'll
	// figure out which fields are actually needed in OutgoingRequest and which are
	// not. Thus please keep the code the way it is for now, dont't introduce any
	// CommonRequestData structures or something like this.
	// This being said the implementation of Request interface differs for Incoming and
	// OutgoingRequest. See corresponding implementation of the interface methods.
	apiReqID := outgoing.APIRequestID

	if outgoing.ReturnMode == record.ReturnSaga {
		apiReqID += fmt.Sprintf("-saga-%d", outgoing.Nonce)
	}

	incoming := record.IncomingRequest{
		Caller:          outgoing.Caller,
		CallerPrototype: outgoing.CallerPrototype,
		Nonce:           outgoing.Nonce,

		Immutable:  outgoing.Immutable,
		ReturnMode: outgoing.ReturnMode,

		CallType:  outgoing.CallType, // used only for CTSaveAsChild
		Base:      outgoing.Base,     // used only for CTSaveAsChild
		Object:    outgoing.Object,
		Prototype: outgoing.Prototype,
		Method:    outgoing.Method,
		Arguments: outgoing.Arguments,

		APIRequestID: apiReqID,
		Reason:       outgoing.Reason,
	}

	return &incoming
}
