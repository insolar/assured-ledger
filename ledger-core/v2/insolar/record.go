// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insolar

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

const (
	// RecordHashSize is a record hash size. We use 224-bit SHA-3 hash (28 bytes).
	RecordHashSize = 28
	// RecordIDSize is relative record address.
	RecordIDSize = PulseNumberSize + RecordHashSize
	// RecordHashOffset is a offset where hash bytes starts in ID.
	RecordHashOffset = PulseNumberSize
	// RecordRefSize is absolute records address (including domain ID).
	RecordRefSize = RecordIDSize * 2
)

type (
	// Reference is a unified record reference
	Reference = reference.Global
)

// IsObjectReferenceString checks the validity of the reference
// deprecated
func IsObjectReferenceString(input string) bool {
	_, err := reference.GlobalObjectFromString(input)
	return err == nil
}
