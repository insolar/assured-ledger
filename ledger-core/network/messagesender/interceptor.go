package messagesender

import (
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
)

type InterceptorFn func(record rmsreg.GoGoSerializable) ([]rmsreg.GoGoSerializable, bool)
