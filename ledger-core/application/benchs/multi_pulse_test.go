// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package benchs

import (
	"context"
	"fmt"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/application/testutils/launchnet"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
)

func BenchmarkMultiPulse(b *testing.B) {
	instestlogger.SetTestOutput(b)

	ctx := context.Background()
	for numNodes := 2; numNodes <= 5; numNodes++ {
		b.Run(fmt.Sprintf("Nodes %d", numNodes), func(b *testing.B) {

			res := launchnet.CustomRunWithPulsar(numNodes, 0, 0, func(apiAddresses []string) int {
				setAPIAddresses(apiAddresses)

				numWallets := numNodes * 10

				wallets := make([]string, 0, numWallets)
				for i := 0; i < numWallets; i++ {
					wallet, err := createSimpleWallet(ctx)
					if err != nil {
						return 2
					}
					wallets = append(wallets, wallet)
				}

				res := runGetBench(wallets)
				if res != nil {
					return 2
				}
				res = runSetBench(wallets)
				if res != nil {
					return 2
				}
				return 0
			})

			if res != 0 {
				b.Error("network run failed")
				b.Fatal("failed test run")
			}
		})
	}
}
