// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package preservedstatereport

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	payload "github.com/insolar/assured-ledger/ledger-core/rms"
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
}

func BuildReportKey(object reference.Global) ReportKey {
	return ReportKey{
		ObjectReference: object,
	}
}

func GetSharedStateReport(ctx smachine.InOrderStepContext, object reference.Global) (SharedReportAccessor, bool) {
	if v := ctx.GetPublishedLink(BuildReportKey(object)); v.IsAssignableTo((*payload.VStateReport)(nil)) {
		return SharedReportAccessor{v}, true
	}
	return SharedReportAccessor{}, false
}
