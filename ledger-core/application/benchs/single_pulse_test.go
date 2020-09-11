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
	"github.com/insolar/assured-ledger/ledger-core/application/testwalletapi/statemachine"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func BenchmarkSinglePulse(b *testing.B) {
	instestlogger.SetTestOutput(b)

	for numNodes := 2; numNodes <= 5; numNodes++ {
		b.Run(fmt.Sprintf("Nodes %d", numNodes), func(b *testing.B) {
			res := launchnet.CustomRunWithoutPulsar(numNodes, 0, 0, runBenchOnNetwork(b, numNodes))
			if res != 0 {
				b.Error("network run failed")
				b.Fatal("failed test run")
			}
		})
	}
}

func runBenchOnNetwork(b *testing.B, numNodes int) func(apiAddresses []string, pulseTime int) int {
	return func(apiAddresses []string, pulseTime int) int {
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

		err := runBriefEchoBench(pulseTime)
		if err != nil {
			b.Error(err)
			return 2
		}

		err = runEchoBench(pulseTime)
		if err != nil {
			b.Error(err)
			return 2
		}

		err = runGetBench(wallets, pulseTime)
		if err != nil {
			b.Error(err)
			return 2
		}
		err = runSetBench(wallets, pulseTime)
		if err != nil {
			b.Error(err)
			return 2
		}
		return 0
	}
}

func runBriefEchoBench(pulseTime int) error {
	return runBenchTimed("brief echo", func(ctx context.Context, iterator int) error {
		walletRef := statemachine.BuiltinTestAPIBriefEcho
		getBalanceURL := getURL(walletGetBalancePath, "")

		_, err := getWalletBalance(ctx, getBalanceURL, walletRef)

		return err
	}, pulseTime)
}

func runEchoBench(pulseTime int) error {
	return runBenchTimed("echo", func(ctx context.Context, iterator int) error {
		walletRef := statemachine.BuiltinTestAPIEcho
		getBalanceURL := getURL(walletGetBalancePath, "")

		_, err := getWalletBalance(ctx, getBalanceURL, walletRef)

		return err
	}, pulseTime)
}

func runSetBench(wallets []string, pulseTime int) error {
	return runBenchTimed("set", func(ctx context.Context, iterator int) error {
		walletRef := wallets[iterator%len(wallets)]
		addAmountURL := getURL(walletAddAmountPath, "")

		return addAmountToWallet(ctx, addAmountURL, walletRef, 1000)
	}, pulseTime)

}

func runGetBench(wallets []string, pulseTime int) error {
	return runBenchTimed("get", func(ctx context.Context, iterator int) error {
		walletRef := wallets[iterator%len(wallets)]
		getBalanceURL := getURL(walletGetBalancePath, "")

		_, err := getWalletBalance(ctx, getBalanceURL, walletRef)

		return err
	}, pulseTime)
}

func getTimeToExecute(pulseTime int) int {
	var (
		minimumExecTime = 10 // seconds
		numPulses       = 2
	)
	timeToRun := minimumExecTime
	pulseTime = pulseTime / 1000 // convert to seconds

	pulseTime = numPulses * pulseTime
	if timeToRun < pulseTime {
		timeToRun = pulseTime
	}

	return timeToRun
}

func runBenchTimed(name string, workerFunc func(ctx context.Context, iterator int) error, pulseTime int) error {
	fmt.Println("==== Running " + name)
	// default Parallelism will be equal to NumCPU
	g, ctx := errgroup.WithContext(context.Background())

	var (
		counter           = ratecounter.NewRateCounter(60 * time.Second)
		timingCounter     = ratecounter.NewAvgRateCounter(60 * time.Second)
		secondRateCounter = ratecounter.NewRateCounter(time.Second)
	)

	startBench := time.Now()
	counterStopped := make(chan struct{})

	// I don't use ticker, because with a lot of goroutines it cannot handle the case
	go func() {
		defer close(counterStopped)
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

	timeFinished := make(chan struct{})
	go func() {
		timeToRun := getTimeToExecute(pulseTime)
		time.Sleep(time.Duration(timeToRun) * time.Second)
		close(timeFinished)
	}()

	canRun := true
	for i := 0; canRun; i++ {
		g.Go(func() error {
			select {
			case <-ctx.Done():
				return nil
			default:
			}

			startTime := time.Now()

			err := workerFunc(ctx, i)

			defer func() {
				timingCounter.Incr(time.Since(startTime).Nanoseconds())
				counter.Incr(1)
				secondRateCounter.Incr(1)
			}()

			if err == context.Canceled {
				return nil
			}

			return err
		})

		select {
		// prevent extra requests if group context is cancelled
		case <-ctx.Done():
			canRun = false
		case <-timeFinished:
			canRun = false
		default:
		}

	}

	err := g.Wait()
	if err != nil {
		return throw.W(err, "api failed")
	}

	finished := time.Since(startBench)

	<-counterStopped

	fmt.Printf("Run took %.02f seconds\n", finished.Seconds())

	if finished.Seconds() > 60 {
		finished = 60 * time.Second
	}

	fmt.Printf("scenario=%s: %d req/s, avg time %.0f ns, num reqs: %d\n",
		name,
		counter.Rate()*int64(time.Second)/finished.Nanoseconds(),
		timingCounter.Rate(),
		counter.Rate(),
	)

	return nil
}
