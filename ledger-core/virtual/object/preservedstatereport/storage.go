package preservedstatereport

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

type SharedReportAccessor struct {
	smachine.SharedDataLink
}

func (v SharedReportAccessor) Prepare(fn func(report rms.VStateReport)) smachine.SharedDataAccessor {
	return v.PrepareAccess(func(data interface{}) bool {
		fn(*data.(*rms.VStateReport))
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
	if v := ctx.GetPublishedLink(BuildReportKey(object)); v.IsAssignableTo((*rms.VStateReport)(nil)) {
		return SharedReportAccessor{v}, true
	}
	return SharedReportAccessor{}, false
}
