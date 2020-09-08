// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package execution

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/runner/call"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
)

type Context struct {
	ObjectDescriptor descriptor.Object
	Context          context.Context
	Request          *rms.VCallRequest
	Result           *rms.VCallResult
	Sequence         uint32
	Pulse            pulse.Data

	Object   reference.Global
	Incoming reference.Global
	Outgoing reference.Global

	Isolation contract.MethodIsolation

	LogicContext call.LogicContext
}
