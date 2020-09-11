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

	"github.com/insolar/assured-ledger/ledger-core/application/testwalletapi/statemachine"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type benchRunner struct {
	benchLimiter benchLimiter
	n            int
}

func (br benchRunner) runBenchOnNetwork(b *testing.B, numNodes int) func(apiAddresses []string) int {
	return func(apiAddresses []string) int {
		b.StartTimer()

		ctx := context.Background()
		setAPIAddresses(apiAddresses)

		numWallets := numNodes * 10 * br.n

		wallets := make([]string, 0, numWallets)
		for i := 0; i < numWallets; i++ {
			wallet, err := createSimpleWallet(ctx)
			if err != nil {
				b.Error(err)
				return 2
			}
			wallets = append(wallets, wallet)
		}

		br.benchLimiter.Reset()
		err := runBriefEchoBench(br.benchLimiter)
		if err != nil {
			b.Error(err)
			return 2
		}

		br.benchLimiter.Reset()
		err = runEchoBench(br.benchLimiter)
		if err != nil {
			b.Error(err)
			return 2
		}

		br.benchLimiter.Reset()
		err = runGetBench(wallets, br.benchLimiter)
		if err != nil {
			b.Error(err)
			return 2
		}

		br.benchLimiter.Reset()
		err = runSetBench(wallets, br.benchLimiter)
		if err != nil {
			b.Error(err)
			return 2
		}

		return 0
	}
}

func runBriefEchoBench(limiter benchLimiter) error {
	return runBench("brief echo", func(ctx context.Context, iterator int) error {
		walletRef := statemachine.BuiltinTestAPIBriefEcho
		getBalanceURL := getURL(walletGetBalancePath, "")

		_, err := getWalletBalance(ctx, getBalanceURL, walletRef)

		return err
	}, limiter)
}

func runEchoBench(limiter benchLimiter) error {
	return runBench("echo", func(ctx context.Context, iterator int) error {
		walletRef := statemachine.BuiltinTestAPIEcho
		getBalanceURL := getURL(walletGetBalancePath, "")

		_, err := getWalletBalance(ctx, getBalanceURL, walletRef)

		return err
	}, limiter)
}

func runSetBench(wallets []string, limiter benchLimiter) error {
	return runBench("set", func(ctx context.Context, iterator int) error {
		walletRef := wallets[iterator%len(wallets)]
		addAmountURL := getURL(walletAddAmountPath, "")

		return addAmountToWallet(ctx, addAmountURL, walletRef, 1000)
	}, limiter)

}

func runGetBench(wallets []string, limiter benchLimiter) error {
	return runBench("get", func(ctx context.Context, iterator int) error {
		walletRef := wallets[iterator%len(wallets)]
		getBalanceURL := getURL(walletGetBalancePath, "")

		_, err := getWalletBalance(ctx, getBalanceURL, walletRef)

		return err
	}, limiter)
}

func runBench(name string, workerFunc func(ctx context.Context, iterator int) error, limiter benchLimiter) error {
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
		reportTicker := time.NewTicker(time.Second)
		defer close(counterStopped)
		for {
			select {
			case <-ctx.Done():
				return
			case <-reportTicker.C:
				fmt.Printf("one second rate %d req/s\n", secondRateCounter.Rate())
			default:
			}
		}
	}()

	stopped := false
	for i := 0; !stopped && limiter.ShouldContinue(); i++ {
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
			stopped = true
		default:
		}

	}

	err := g.Wait()
	if err != nil {
		return throw.W(err, "api failed")
	}

	finished := time.Since(startBench)

	//<-counterStopped

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
