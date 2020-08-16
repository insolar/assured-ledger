// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package benchs

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/neilotoole/errgroup"
	"github.com/paulbellamy/ratecounter"

	"github.com/insolar/assured-ledger/ledger-core/application/testutils/launchnet"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
)

func BenchmarkSinglePulse(b *testing.B) {
	instestlogger.SetTestOutput(b)

	for numNodes := 2; numNodes <= 5; numNodes++ {
		b.Run(fmt.Sprintf("Nodes %d", numNodes), func(b *testing.B) {
			res := launchnet.CustomRunWithoutPulsar(numNodes, 0, 0, runBenchOnNetwork(numNodes))
			if res != 0 {
				b.Error("network run failed")
				b.Fatal("failed test run")
			}
		})
	}
}

func runBenchOnNetwork(numNodes int) func(apiAddresses []string) int {
	return func(apiAddresses []string) int {
		ctx := context.Background()
		setAPIAddresses(apiAddresses)

		numWallets := numNodes * 1000

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
	}
}

func runGetBench(wallets []string) error {
	fmt.Println("==== Get run")

	// default Parallelism will be equal to NumCPU
	g, ctx := errgroup.WithContext(context.Background())

	counter := ratecounter.NewRateCounter(60 * time.Second)
	timingCounter := ratecounter.NewAvgRateCounter(60 * time.Second)
	startBench := time.Now()

	for i := 1; i < 100000; i++ {
		// prevent extra requests if group context is cancelled
		select {
		case <-ctx.Done():
			break
		default:
		}
		g.Go(func() error {
			walletRef := wallets[i%len(wallets)]
			getBalanceURL := getURL(walletGetBalancePath, "")

			startTime := time.Now()

			_, err := getWalletBalance(ctx, getBalanceURL, walletRef)

			timingCounter.Incr(time.Since(startTime).Nanoseconds())
			counter.Incr(1)
			return err
		})
	}

	finished := time.Since(startBench)

	finished.Round(time.Second)

	fmt.Printf("Run took %d seconds\n", int(finished.Seconds()))

	if finished.Seconds() > 60 {
		finished = 60 * time.Second
	}

	fmt.Printf("get rate %d req/s, avg time %.0f ns\n", counter.Rate()/int64(finished.Seconds()), timingCounter.Rate())

	return g.Wait()
}

func runSetBench(wallets []string) error {
	fmt.Println("==== Set run")

	// default Parallelism will be equal to NumCPU
	g, ctx := errgroup.WithContext(context.Background())

	counter := ratecounter.NewRateCounter(60 * time.Second)
	timingCounter := ratecounter.NewAvgRateCounter(60 * time.Second)
	startBench := time.Now()

	for i := 1; i < 100000; i++ {
		g.Go(func() error {
			// prevent extra requests if group context is cancelled
			select {
			case <-ctx.Done():
				break
			default:
			}
			walletRef := wallets[i%len(wallets)]
			addAmountURL := getURL(walletAddAmountPath, "")

			startTime := time.Now()

			err := addAmountToWallet(ctx, addAmountURL, walletRef, 1000)

			timingCounter.Incr(time.Since(startTime).Nanoseconds())
			counter.Incr(1)

			return err
		})
	}

	finished := time.Since(startBench)

	finished.Round(time.Second)

	fmt.Printf("Run took %d seconds\n", int(finished.Seconds()))

	if finished.Seconds() > 60 {
		finished = 60 * time.Second
	}

	fmt.Printf("set rate %d req/s, avg time %.0f ns\n", counter.Rate()/int64(finished.Seconds()), timingCounter.Rate())

	return g.Wait()
}