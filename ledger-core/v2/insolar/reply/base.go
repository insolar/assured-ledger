// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// Package reply represents responses to messages of the messagebus
package reply

import (
	"encoding/gob"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
)

const (
	// Generic

	// TypeError is reply with error.
	TypeError = insolar.ReplyType(iota + 1)
	// TypeOK is a generic reply for signaling a positive result.
	TypeOK
	// TypeNotOK is a generic reply for signaling a negative result.
	TypeNotOK

	// Logicrunner

	// TypeCallMethod - two binary fields: data and results.
	TypeCallMethod
	// TypeRegisterRequest - request for execution was registered
	TypeRegisterRequest
)

// ErrType is used to determine and compare reply errors.
type ErrType int

const (
	// ErrDeactivated returned when requested object is deactivated.
	ErrDeactivated = iota + 1
	// ErrStateNotAvailable is returned when requested object is deactivated.
	ErrStateNotAvailable
	// ErrHotDataTimeout is returned when no hot data received for a specific jet
	ErrHotDataTimeout
	// ErrNoPendingRequests is returned when there are no pending requests on current LME
	ErrNoPendingRequests
	// FlowCancelled is returned when a new pulse happened in the process of message execution
	FlowCancelled
)

func init() {
	gob.Register(&CallMethod{})
	gob.Register(&RegisterRequest{})
	gob.Register(&Error{})
	gob.Register(&OK{})
}
