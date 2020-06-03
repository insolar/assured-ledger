// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package testutils

import (
	"go.uber.org/goleak"
)

func LeakTester(t goleak.TestingT, extraOpts ...goleak.Option) {
	extraOpts = append(extraOpts, goleak.IgnoreTopFunction("runtime/pprof.readProfile"),
		goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"),
		// sometimes stack has full import path
		goleak.IgnoreTopFunction("github.com/insolar/insolar/vendor/go.opencensus.io/stats/view.(*worker).start"),
		goleak.IgnoreTopFunction("github.com/insolar/insolar/log/critlog.(*internalBackpressureBuffer).worker"),
		goleak.IgnoreTopFunction("github.com/insolar/assured-ledger/ledger-core/log/bpbuffer.(*internalBackpressureBuffer).worker"),
		goleak.IgnoreTopFunction("github.com/insolar/assured-ledger/ledger-core/vanilla/synckit/SignalVersion.waitClose"))
	goleak.VerifyNone(t,
		extraOpts...)
}
