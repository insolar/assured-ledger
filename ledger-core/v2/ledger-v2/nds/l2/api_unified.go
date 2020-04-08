// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l2

type PayloadLengthLimit uint8

const (
	_ PayloadLengthLimit = iota
	DetectByFirstPayloadLength
	NonExcessivePayloadLength
	UnlimitedPayloadLength
)
