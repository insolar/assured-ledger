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
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
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

func runSetBench(wallets []string) error {
	return runBench(10000, "set", func(ctx context.Context, iterator int) error {
		walletRef := wallets[iterator%len(wallets)]
		addAmountURL := getURL(walletAddAmountPath, "")

		return addAmountToWallet(ctx, addAmountURL, walletRef, 1000)
	})
}

func runGetBench(wallets []string) error {
	return runBench(10000, "get", func(ctx context.Context, iterator int) error {
		walletRef := wallets[iterator%len(wallets)]
		getBalanceURL := getURL(walletGetBalancePath, "")

		_, err := getWalletBalance(ctx, getBalanceURL, walletRef)

		return err
	})
}

func runBench(iterations int, name string, workerFunc func(ctx context.Context, iterator int) error) error {
	fmt.Println("==== Running " + name)
	// default Parallelism will be equal to NumCPU
	g, ctx := errgroup.WithContext(context.Background())

	var (
		counter           = ratecounter.NewRateCounter(60 * time.Second)
		timingCounter     = ratecounter.NewAvgRateCounter(60 * time.Second)
		secondRateCounter = ratecounter.NewRateCounter(time.Second)
	)

	startBench := time.Now()

	// I don't use ticker, because with a lot of goroutines it cannot handle the case
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			time.Sleep(time.Second)
			fmt.Printf("one second rate %d req/s\n", secondRateCounter.Rate())
		}
	}()

	for i := 1; i <= iterations; i++ {
		// prevent extra requests if group context is cancelled
		select {
		case <-ctx.Done():
			break
		default:
		}

		g.Go(func() error {
			startTime := time.Now()

			err := workerFunc(ctx, i)

			timingCounter.Incr(time.Since(startTime).Nanoseconds())
			counter.Incr(1)
			secondRateCounter.Incr(1)

			return err
		})
	}

	err := g.Wait()
	if err != nil {
		return throw.W(err, "api failed")
	}

	finished := time.Since(startBench)

	finished.Round(time.Second)

	fmt.Printf("Run took %d seconds\n", int(finished.Seconds()))

	if finished.Seconds() > 60 {
		finished = 60 * time.Second
	}

	fmt.Printf("scenario=%s: %d req/s, avg time %.0f ns\n", name, counter.Rate()/int64(finished.Seconds()), timingCounter.Rate())

	return nil
}
