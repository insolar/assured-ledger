// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type CallRequestFlags uint32

func (f CallRequestFlags) Equal(r CallRequestFlags) bool {
	return f == r
}

type SendResultFullFlag byte

const (
	SendResultDefault SendResultFullFlag = iota
	SendResultFull
)

func (f SendResultFullFlag) IsZero() bool {
	return f == 0
}

type RepeatedCallFlag byte

const (
	CallDefault RepeatedCallFlag = iota
	RepeatedCall
)

func (f RepeatedCallFlag) IsZero() bool {
	return f == 0
}

const (
	bitSendResultFullFlagCount = 1
	bitRepeatedCallFlagCount   = 1

	bitSendResultFullOffset = 0
	bitRepeatedCallOffset   = bitSendResultFullOffset + bitSendResultFullFlagCount
)

const (
	bitSendResultFullMask = ((1 << bitSendResultFullFlagCount) - 1) << bitSendResultFullOffset
)

func (f CallRequestFlags) WithSendResultFull(t SendResultFullFlag) CallRequestFlags {
	if t > bitSendResultFullFlagCount {
		panic(throw.IllegalValue())
	}
	return (f &^ bitSendResultFullMask) | (CallRequestFlags(t) << bitSendResultFullOffset)
}

func (f CallRequestFlags) GetSendResult() SendResultFullFlag {
	return SendResultFullFlag(f&bitSendResultFullMask) >> bitSendResultFullOffset
}

const (
	bitRepeatedCallMask = ((1 << bitRepeatedCallFlagCount) - 1) << bitRepeatedCallOffset
)

func (f CallRequestFlags) WithRepeatedCall(s RepeatedCallFlag) CallRequestFlags {
	if s > bitRepeatedCallFlagCount {
		panic(throw.IllegalValue())
	}
	return (f &^ bitRepeatedCallMask) | (CallRequestFlags(s) << bitRepeatedCallOffset)
}

func (f CallRequestFlags) GetRepeatedCall() RepeatedCallFlag {
	return RepeatedCallFlag(f&bitRepeatedCallMask) >> bitRepeatedCallOffset
}

func BuildCallRequestFlags(sendResultFull SendResultFullFlag, repeatedCall RepeatedCallFlag) CallRequestFlags {

	return CallRequestFlags(0).WithSendResultFull(sendResultFull).WithRepeatedCall(repeatedCall)
}
