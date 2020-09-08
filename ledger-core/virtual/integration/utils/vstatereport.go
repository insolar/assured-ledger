// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package utils

import (
	testwalletProxy "github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

func GenerateVStateReport(server *Server, object rms.Reference, pulse rms.PulseNumber) *rms.VStateReport {
	content := &rms.VStateReport_ProvidedContentBody{
		LatestDirtyState: &rms.ObjectState{
			Reference: reference.Local{},
			Class:     testwalletProxy.GetClass(),
			State:     []byte("dirty"),
		},
		LatestValidatedState: &rms.ObjectState{
			Reference: reference.Local{},
			Class:     testwalletProxy.GetClass(),
			State:     []byte("validated"),
		},
	}
	return &rms.VStateReport{
		Status:          rms.StateStatusReady,
		Object:          object,
		AsOf:            pulse,
		ProvidedContent: content,
	}
}
