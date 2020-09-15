// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package benchs

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/neilotoole/errgroup"
	"github.com/paulbellamy/ratecounter"

	"github.com/insolar/assured-ledger/ledger-core/application/testwalletapi/statemachine"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type benchRunner struct {
	benchLimiterFactory limiterFactory
}

var (
	errCode int32 = 2
)

func (br benchRunner) runBenchOnNetwork(b *testing.B) func(apiAddresses []string) int {
	return func(apiAddresses []string) int {
		ctx := context.Background()
		setAPIAddresses(apiAddresses)

		numWallets := 1000
		wallets := make([]string, 0, numWallets)
		for i := 0; i < numWallets; i++ {
			wallet, err := createSimpleWallet(ctx)
			if err != nil {
				b.Error(err)
				return int(errCode)
			}
			wallets = append(wallets, wallet)
		}

		retCode := runBriefEchoBench(b, br.benchLimiterFactory)
		if retCode != 0 {
			return retCode
		}

		retCode = runEchoBench(b, br.benchLimiterFactory)
		if retCode != 0 {
			return retCode
		}

		retCode = runGetBench(wallets, b, br.benchLimiterFactory)
		if retCode != 0 {
			return retCode
		}

		retCode = runSetBench(wallets, b, br.benchLimiterFactory)
		if retCode != 0 {
			return retCode
		}

		return retCode
	}
}

func runBriefEchoBench(b *testing.B, limiterFactory limiterFactory) int {
	var retCode int32
	name := "brief echo"
	b.Run(name, func(b *testing.B) {
		limiter := limiterFactory(b.N)
		err := runBench(name, func(ctx context.Context, iterator int) error {
			walletRef := statemachine.BuiltinTestAPIBriefEcho
			getBalanceURL := getURL(walletGetBalancePath, "")

			_, err := getWalletBalance(ctx, getBalanceURL, walletRef)

			return err
		}, limiter)

		if err != nil {
			b.Error(err)
			atomic.StoreInt32(&retCode, errCode)
			return
		}
	})

	return int(retCode)
}

func runEchoBench(b *testing.B, limiterFactory limiterFactory) int {
	name := "echo"
	var retCode int32
	b.Run(name, func(b *testing.B) {
		limiter := limiterFactory(b.N)
		err := runBench(name, func(ctx context.Context, iterator int) error {
			walletRef := statemachine.BuiltinTestAPIEcho
			getBalanceURL := getURL(walletGetBalancePath, "")

			_, err := getWalletBalance(ctx, getBalanceURL, walletRef)

			return err
		}, limiter)
		if err != nil {
			b.Error(err)
			atomic.StoreInt32(&retCode, 2)
			return
		}

	})
	return int(retCode)
}

func runSetBench(wallets []string, b *testing.B, limiterFactory limiterFactory) int {
	name := "set"
	var retCode int32
	b.Run(name, func(b *testing.B) {
		limiter := limiterFactory(b.N)
		err := runBench("set", func(ctx context.Context, iterator int) error {
			walletRef := wallets[iterator%len(wallets)]
			addAmountURL := getURL(walletAddAmountPath, "")

			return addAmountToWallet(ctx, addAmountURL, walletRef, 1000)
		}, limiter)
		if err != nil {
			b.Error(err)
			atomic.StoreInt32(&retCode, 2)
			return
		}

	})
	return int(retCode)

}

func runGetBench(wallets []string, b *testing.B, limiterFactory limiterFactory) int {
	name := "get"
	var retCode int32
	b.Run(name, func(b *testing.B) {
		limiter := limiterFactory(b.N)
		err := runBench(name, func(ctx context.Context, iterator int) error {
			walletRef := wallets[iterator%len(wallets)]
			getBalanceURL := getURL(walletGetBalancePath, "")

			_, err := getWalletBalance(ctx, getBalanceURL, walletRef)

			return err
		}, limiter)
		if err != nil {
			b.Error(err)
			atomic.StoreInt32(&retCode, 2)
			return
		}

	})
	return int(retCode)
}

func runBench(name string, workerFunc func(ctx context.Context, iterator int) error, limiter benchLimiter) error {
	// default Parallelism will be equal to NumCPU
	g, ctx := errgroup.WithContext(context.Background())

	var (
		counter           = ratecounter.NewRateCounter(60 * time.Second)
		timingCounter     = ratecounter.NewAvgRateCounter(60 * time.Second)
		secondRateCounter = ratecounter.NewRateCounter(time.Second)
	)

	startBench := time.Now()
	counterStopped := make(chan struct{})

	if limiter.NeedPrintStats() {
		go func() {
			defer close(counterStopped)
			reportTicker := time.NewTicker(time.Second)
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
	}

	limiter.Reset()
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

	if finished.Seconds() > 60 {
		finished = 60 * time.Second
	}

	if limiter.NeedPrintStats() {
		fmt.Printf("scenario=%s: %d req/s, avg time %.0f ns, num reqs: %d\n",
			name,
			counter.Rate()*int64(time.Second)/finished.Nanoseconds(),
			timingCounter.Rate(),
			counter.Rate(),
		)
	}

	if limiter.NeedPrintStats() {
		<-counterStopped
	}

	return nil
}
