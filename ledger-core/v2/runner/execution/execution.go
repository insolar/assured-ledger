// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package execution

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/call"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
)

// TODO[bigbes]: redo context, extract what is needed from VCallRequest to Context level and etc
type Context struct {
	ObjectDescriptor descriptor.Object
	Context          context.Context
	Request          *payload.VCallRequest
	Sequence         uint32
	Pulse            pulse.Data

	Object   reference.Global
	Incoming reference.Global
	Outgoing reference.Global

	Isolation contract.MethodIsolation

	LogicContext call.LogicContext
}
