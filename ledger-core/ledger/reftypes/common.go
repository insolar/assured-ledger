// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reftypes

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var ErrInvalidRef = throw.E("invalid reference")
var ErrEmptyRef = throw.E("empty reference")
var ErrNotEmptyRef = throw.E("not empty reference")
var ErrIllegalRefValue = throw.E("illegal reference value")
var ErrIllegalSelfRefValue = throw.W(ErrIllegalRefValue,"self-reference expected")

func pulseZeroScope(h reference.LocalHeader) pulse.Number {
	if h.SubScope() == 0 {
		return h.Pulse()
	}
	return pulse.Unknown
}

type DetailErrRef struct {
	Expected RefType
	BaseHeader, LocalHeader reference.LocalHeader
}

func newRefTypeErr(err error, expected RefType, base, local reference.Local) error {
	return throw.WithDetails(err, DetailErrRef { expected, base.GetHeader(), local.GetHeader() })
}

