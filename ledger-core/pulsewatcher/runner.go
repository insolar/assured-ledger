// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulsewatcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/application/api/requester"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/pulsewatcher/output"
	jsonOutput "github.com/insolar/assured-ledger/ledger-core/pulsewatcher/output/json"
	"github.com/insolar/assured-ledger/ledger-core/pulsewatcher/output/text"
	"github.com/insolar/assured-ledger/ledger-core/pulsewatcher/status"
)

func CollectNodesStatuses(cfg configuration.PulseWatcherConfig, lastResults []status.Node) []status.Node {
	results := make([]status.Node, len(cfg.Nodes))
	lock := &sync.Mutex{}

	client := http.Client{
		Transport: &http.Transport{},
		Timeout:   cfg.Timeout,
	}

	wg := &sync.WaitGroup{}
	wg.Add(len(cfg.Nodes))
	for i, url := range cfg.Nodes {
		go func(url string, i int) {
			res, err := client.Post("http://"+url+"/admin-api/rpc", "application/json",
				strings.NewReader(`{"jsonrpc": "2.0", "method": "node.getStatus", "id": 0}`))

			url = strings.TrimPrefix(url, "127.0.0.1")

			if err != nil {
				errStr := err.Error()
				if strings.Contains(errStr, "connection refused") ||
					strings.Contains(errStr, "request canceled while waiting for connection") ||
					strings.Contains(errStr, "no such host") {
					// Print compact error string when node is down.
					// This prevents table distortion on small screens.
					errStr = "NODE IS DOWN"
				}
				if strings.Contains(errStr, "exceeded while awaiting headers") {
					errStr = "TIMEOUT"
				}

				lock.Lock()
				if len(lastResults) > i {
					results[i] = lastResults[i]
					results[i].ErrStr = errStr
				} else {
					results[i] = status.Node{url, requester.StatusResponse{}, errStr}
				}
				lock.Unlock()
				wg.Done()
				return
			}
			defer res.Body.Close()
			data, err := ioutil.ReadAll(res.Body)
			if err != nil {
				global.Fatal(err)
			}
			var out struct {
				Result requester.StatusResponse
			}
			err = json.Unmarshal(data, &out)
			if err != nil {
				fmt.Println(string(data))
				log.Fatal(err)
			}
			lock.Lock()

			results[i] = status.Node{url, out.Result, ""}

			lock.Unlock()
			wg.Done()
		}(url, i)
	}
	wg.Wait()

	return results
}

func Run(_ context.Context, cfg configuration.PulseWatcherConfig) {
	var (
		results []status.Node
		printer output.Printer
	)
	switch cfg.Format {
	case configuration.PulseWatcherOutputJson:
		printer = jsonOutput.NewOutput()
	case configuration.PulseWatcherOutputTxt:
		printer = text.NewOutput(cfg)
	default:
		global.Fatal("Unhandled output format:" + string(cfg.Format))
	}

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()
	for ; true; <-ticker.C {
		results = CollectNodesStatuses(cfg, results)
		printer.Print(results)

		if cfg.OneShot {
			break
		}
	}
}
