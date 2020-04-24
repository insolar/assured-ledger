// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package execution

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
)

// TODO[bigbes]: redo context, extract what is needed from VCallRequest to Context level and etc
type Context struct {
	ObjectDescriptor descriptor.ObjectDescriptor
	Context          context.Context
	Request          *payload.VCallRequest
	Sequence         uint32
	Pulse            pulse.Data

	Object   insolar.Reference
	Incoming insolar.Reference
	Outgoing insolar.Reference

	Unordered bool

	LogicContext insolar.LogicCallContext
}
