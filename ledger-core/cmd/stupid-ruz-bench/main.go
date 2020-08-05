// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"

	"github.com/insolar/assured-ledger/ledger-core/application/testwalletapi"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
)

var baseURL = "http://127.0.0.1"
var ports = []string{"32301","32302","32303","32304","32305"}
var jsonCodec = jsoniter.ConfigCompatibleWithStandardLibrary

func main() {
	global.SetLevel(log.ErrorLevel)

	refs := createWallets(100)
	println("created wallet: ", refs[0])

	ref := refs[rand.Intn(len(refs))]
	println("balance: ", getBalance(ref))

	baseRPS := 0.0
	for conc := 1; conc <= 101; conc+=20 {
		count, took := getBalances(refs, 15*time.Second, conc)

		rps := rps(count, took)
		ratio := 1.0
		scalingFactor := 1.0
		if conc == 1 {
			baseRPS = rps
		} else {
			ratio = rps/baseRPS
			scalingFactor = ratio/float64(conc)
		}
		fmt.Printf(
			"concurrency %5d: %10.2frps %6.2fx %6.2fsf (%d in %.4fs)\n",
			conc, rps, ratio, scalingFactor, count, took.Seconds(),
		)
	}
}


func createWallet() string {
	result := testwalletapi.TestWalletServerCreateResult{}
	post("create", strings.NewReader("{}"), &result)

	if result.Error != "" {
		check("couldn't create", errors.New(result.Error))
	}

	return result.Reference
}

func createWallets(num int) []string {
	res := make([]string, num)
	for i := 0; i < num; i++ {
		res[i] = createWallet()
	}
	return res
}

func getBalance(ref string) uint {

	args, err := jsonCodec.MarshalToString(testwalletapi.GetBalanceParams{WalletRef: ref})
	check("couldn't marshal json", err)

	result := testwalletapi.TestWalletServerGetBalanceResult{}

	start := time.Now()
	post("get_balance", strings.NewReader(args), &result)
	spent := time.Since(start)

	if result.Error != "" {
		check("couldn't get balance", errors.New(result.Error))
	}

	if spent.Seconds() > 1.0 {
		check("slow", errors.New(fmt.Sprintf("getBalance '%s' took %0.4f", result.TraceID, spent.Seconds())))
	}

	return result.Amount
}

func getBalances(refs []string, dur time.Duration, concurency int) (int, time.Duration) {
	refsLen := len(refs)

	sig := make(chan struct{}, 0)
	counter := atomickit.Int{}
	wg := sync.WaitGroup{}
	wg.Add(concurency)

	start := time.Now()

	for i:= 0; i<concurency; i++ {
		go func() {
			defer wg.Done()

			for {
				getBalance(refs[rand.Intn(refsLen)])
				counter.Add(1)
				select {
				case <-sig:
					return
				default:
				}
			}
		}()
	}

	<-time.After(dur)
	close(sig)
	wg.Wait()
	timeSpent := time.Since(start)

	return counter.Load(), timeSpent
}

func check(msg string, err error) {
	if err != nil {
		fmt.Println(msg, err)
		os.Exit(1)
	}
}

func randPort() string {
	return ports[rand.Intn(len(ports))]
}

func buildUrl(loc string) string {
	return baseURL + ":" + randPort() + "/wallet/" + loc
}

func parseJson(resp *http.Response, to interface{}) {
	body, err := ioutil.ReadAll(resp.Body)
	check("couldn't read body", err)

	err = jsonCodec.Unmarshal(body, to)
	check("couldn't decode body", err)
}

func post(loc string, args io.Reader, to interface{}) {
	resp, err := http.Post(
		buildUrl(loc), "application/json", args,
	)
	check("couldn't call "+loc, err)
	parseJson(resp, to)
}


func rps(count int, duration time.Duration) float64 {
	seconds := float64(duration)/float64(time.Second)
	return float64(count)/seconds
}
