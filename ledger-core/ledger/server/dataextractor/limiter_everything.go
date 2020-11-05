// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package dataextractor

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

func NewNoLimits() Limiter {
	return everything{}
}

var _ Limiter = everything{}
type everything struct {}

func (everything) Clone() Limiter {
	return NewNoLimits()
}

func (everything) CanRead() bool {
	return true
}

func (everything) IsSkipped(uint32) bool {
	return false
}

func (everything) CanReadExcerpt() bool {
	return true
}

func (everything) CanReadBody() bool {
	return true
}

func (everything) CanReadPayload() bool {
	return true
}

func (everything) CanReadExtensions() bool {
	return true
}

func (everything) CanReadExtension(ledger.ExtensionID) bool {
	return true
}

func (everything) Next(int, reference.Holder) {}
