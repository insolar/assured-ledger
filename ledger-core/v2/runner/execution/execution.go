// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package execution

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
)

type Execution struct {
	Reference  insolar.Reference
	Object     descriptor.ObjectDescriptor
	Context    context.Context
	Request    *payload.VCallRequest
	Sequence   uint32
	Deactivate bool
	Pulse      insolar.Pulse

	LogicContext insolar.LogicCallContext
}
