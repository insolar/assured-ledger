// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package mimic

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/record"
)

func recordStateToStateID(state record.State) record.StateID {
	switch state.(type) {
	case *record.Activate:
		return record.StateActivation
	case *record.Amend:
		return record.StateAmend
	case *record.Deactivate:
		return record.StateDeactivation
	default:
		return record.StateUndefined
	}
}

type ObjectState struct {
	State     record.State
	RequestID insolar.ID
}

type ObjectEntity struct {
	ObjectChanges []ObjectState
	RequestsMap   map[insolar.ID]*RequestEntity
	RequestsList  []*RequestEntity
}

func (e *ObjectEntity) isDeactivated() bool {
	return e.getLatestStateID() == record.StateDeactivation
}

func (e *ObjectEntity) addIncomingRequest(entity *RequestEntity) {
	e.RequestsMap[entity.ID] = entity
	e.RequestsList = append(e.RequestsList, entity)
}

func (e ObjectEntity) getLatestStateID() record.StateID {
	if len(e.ObjectChanges) == 0 {
		return record.StateUndefined
	}

	return recordStateToStateID(e.ObjectChanges[len(e.ObjectChanges)-1].State)
}

// getRequestInfo returns:
// * count of opened requests
// * earliest request
// * latest request
func (e ObjectEntity) getRequestsInfo() (uint32, *RequestEntity, *RequestEntity) {
	var (
		openRequestCount uint32
		firstRequest     *RequestEntity
		lastRequest      *RequestEntity
	)

	for _, request := range e.RequestsList {
		if request.Status == RequestRegistered {
			openRequestCount++
			if firstRequest == nil {
				firstRequest = request
			}
			lastRequest = request
		}
	}
	return openRequestCount, firstRequest, lastRequest
}

type CodeEntity struct {
	Code []byte
}
