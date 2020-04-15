// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package mimic

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/record"
)

type RequestStatus int

const (
	RequestRegistered RequestStatus = iota
	RequestFinished
)

type RequestEntity struct {
	ID        insolar.ID
	Status    RequestStatus
	Request   record.Request
	Outgoings map[insolar.ID]*RequestEntity
	Result    []byte
	ResultID  insolar.ID
}

type outgoingInfo struct {
	requestID insolar.ID
	request   record.Request
}

// TODO[bigbes]: support deduplication here
func (e *RequestEntity) appendOutgoing(outgoingEntity *RequestEntity) {
	if _, ok := outgoingEntity.Request.(*record.OutgoingRequest); !ok {
		panic("Outgoing is not outgoing")
	}
	e.Outgoings[outgoingEntity.ID] = outgoingEntity
}

func (e RequestEntity) hasOpenedOutgoings() bool { // nolint:unused
	for _, req := range e.Outgoings {
		if req.Status != RequestFinished && req.Request.GetReturnMode() != record.ReturnSaga {
			return true
		}
	}

	return false
}

func (e RequestEntity) getSagaOutgoingRequestIDs() []*outgoingInfo {
	var rv []*outgoingInfo
	for _, req := range e.Outgoings {
		if req.Status != RequestFinished && req.Request.GetReturnMode() == record.ReturnSaga {
			rv = append(rv, &outgoingInfo{
				requestID: req.ID,
				request:   req.Request,
			})
		}
	}

	return rv
}

func (e *RequestEntity) getPulse() insolar.PulseNumber {
	return e.ID.Pulse()
}

func NewIncomingRequestEntity(requestID insolar.ID, request record.Request) *RequestEntity {
	_, ok := request.(*record.IncomingRequest)
	if !ok {
		return nil
	}
	return &RequestEntity{
		Status:    RequestRegistered,
		Result:    nil,
		Request:   request.(*record.IncomingRequest),
		ID:        requestID,
		Outgoings: make(map[insolar.ID]*RequestEntity),
	}
}

func NewOutgoingRequestEntity(requestID insolar.ID, request record.Request) *RequestEntity {
	_, ok := request.(*record.OutgoingRequest)
	if !ok {
		return nil
	}
	return &RequestEntity{
		Status:    RequestRegistered,
		Result:    nil,
		Request:   request.(*record.OutgoingRequest),
		ID:        requestID,
		Outgoings: nil,
	}
}
