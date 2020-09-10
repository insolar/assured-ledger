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
			Reference: rms.NewReference(reference.Local{}),
			Class:     rms.NewReference(testwalletProxy.GetClass()),
			State:     rms.NewBytes([]byte("dirty")),
		},
		LatestValidatedState: &rms.ObjectState{
			Reference: rms.NewReference(reference.Local{}),
			Class:     rms.NewReference(testwalletProxy.GetClass()),
			State:     rms.NewBytes([]byte("validated")),
		},
	}
	return &rms.VStateReport{
		Status:          rms.StateStatusReady,
		Object:          rms.NewReference(object),
		AsOf:            pulse,
		ProvidedContent: content,
	}
}
