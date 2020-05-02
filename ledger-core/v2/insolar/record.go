// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insolar

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

// IsObjectReferenceString checks the validity of the reference
// deprecated
func IsObjectReferenceString(input string) bool {
	_, err := reference.GlobalObjectFromString(input)
	return err == nil
}
