package utils

import (
	testwalletProxy "github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

func GenerateVStateReport(server *Server, object payload.Reference, pulse payload.PulseNumber) *payload.VStateReport {
	content := &payload.VStateReport_ProvidedContentBody{
		LatestDirtyState: &payload.ObjectState{
			Reference: reference.Local{},
			Class:     testwalletProxy.GetClass(),
			State:     []byte("dirty"),
		},
		LatestValidatedState: &payload.ObjectState{
			Reference: reference.Local{},
			Class:     testwalletProxy.GetClass(),
			State:     []byte("validated"),
		},
	}
	return &payload.VStateReport{
		Status:          payload.StateStatusReady,
		Object:          object,
		AsOf:            pulse,
		ProvidedContent: content,
	}
}
