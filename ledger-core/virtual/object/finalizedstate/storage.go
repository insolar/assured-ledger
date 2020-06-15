// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package finalizedstate

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type SharedReportAccessor struct {
	smachine.SharedDataLink
}

func (v SharedReportAccessor) Prepare(fn func(report payload.VStateReport)) smachine.SharedDataAccessor {
	return v.PrepareAccess(func(data interface{}) bool {
		fn(*data.(*payload.VStateReport))
		return false
	})
}

type ReportKey struct {
	ObjectReference reference.Global
	Pulse           pulse.Number
}

func BuildReportKey(object reference.Global, pulse pulse.Number) ReportKey {
	return ReportKey{
		ObjectReference: object,
		Pulse:           pulse,
	}
}

func GetSharedStateReport(ctx smachine.InOrderStepContext, object reference.Global, pn pulse.Number) (SharedReportAccessor, bool) {
	if v := ctx.GetPublishedLink(BuildReportKey(object, pn)); v.IsAssignableTo((*payload.VStateReport)(nil)) {
		return SharedReportAccessor{v}, true
	}
	return SharedReportAccessor{}, false
}
