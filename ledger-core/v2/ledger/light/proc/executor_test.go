// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package proc_test

import (
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger/light/proc"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils"
)

func TestCalculateID_Proceed(t *testing.T) {
	ctx := inslogger.TestContext(t)
	mc := minimock.NewController(t)

	var (
		pcs insolar.PlatformCryptographyScheme
	)
	setup := func() {
		pcs = testutils.NewPlatformCryptographyScheme()
	}

	t.Run("happy", func(t *testing.T) {
		setup()
		defer mc.Finish()

		pl, _ := (&payload.Meta{
			Polymorph:  0,
			Payload:    nil,
			Sender:     insolar.Reference{},
			Receiver:   insolar.Reference{},
			Pulse:      0,
			ID:         nil,
			OriginHash: payload.MessageHash{},
		}).Marshal()

		pn := gen.PulseNumber()

		p := proc.NewCalculateID(pl, pn)
		p.Dep(pcs)
		err := p.Proceed(ctx)
		assert.NoError(t, err)
	})
}
