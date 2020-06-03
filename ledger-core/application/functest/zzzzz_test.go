// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build functest

package functest

import (
	"bytes"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/testutils/launchnet"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/insmetrics"
)

// This test file contains tests what always must be last in the package.

// TestMnt_DumpMetrics saves metrics values to files in launchnet logs dir.
func TestMnt_DumpMetrics(t *testing.T) {
	if !launchnet.DumpMetricsEnabled() {
		t.Skip("dump metrics disabled")
	}

	res, err := launchnet.FetchAndSaveMetrics(functestCount - 1)
	if err != nil {
		t.Errorf("metrics save failed: %v", err.Error())
	}
	var inc float64
	for _, b := range res {
		inc += insmetrics.SumMetricsValueByNamePrefix(bytes.NewReader(b), "insolar_requests_abandoned")
	}
	t.Logf("Abandons sum: %v (functestCount=%v)", inc, functestCount)
}
