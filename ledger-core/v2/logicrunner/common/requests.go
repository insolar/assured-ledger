// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package common

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/record"
)

type OutgoingRequest struct {
	Request  record.IncomingRequest
	Response []byte
	Error    error
}
