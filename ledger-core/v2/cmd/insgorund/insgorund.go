// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"context"
	"io/ioutil"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/insolar/assured-ledger/ledger-core/v2/configuration"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/global"
	"github.com/insolar/assured-ledger/ledger-core/v2/metrics"

	"github.com/spf13/pflag"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/goplugin/ginsider"
)

func main() {
	listen := pflag.StringP("listen", "l", ":7777", "address and port to listen")
	protocol := pflag.String("proto", "tcp", "listen protocol")
	path := pflag.StringP("directory", "d", "", "directory where to store code of go plugins")
	rpcAddress := pflag.String("rpc", "localhost:7778", "address and port of RPC API")
	rpcProtocol := pflag.String("rpc-proto", "tcp", "protocol of RPC API")
	metricsAddress := pflag.String("metrics", "", "address and port of prometheus metrics")
	code := pflag.String("code", "", "add pre-compiled code to cache (<ref>:</path/to/plugin.so>)")
	logLevel := pflag.String("log-level", "debug", "log level")

	pflag.Parse()

	err := global.SetTextLevel(*logLevel)
	if err != nil {
		global.Fatalf("Couldn't set log level to %q: %s", *logLevel, err)
	}
	global.InitTicker()

	if *path == "" {
		tmpDir, err := ioutil.TempDir("", "funcTestContractcache-")
		if err != nil {
			global.Fatalf("Couldn't create temp cache dir: %s", err.Error())
		}
		defer func() {
			err := os.RemoveAll(tmpDir)
			if err != nil {
				global.Fatalf("Failed to clean up tmp dir: %s", err.Error())
			}
		}()
		*path = tmpDir
		global.Debug("ginsider cache dir is " + tmpDir)
	}

	insider := ginsider.NewGoInsider(*path, *rpcProtocol, *rpcAddress)

	if *code != "" {
		codeSlice := strings.Split(*code, ":")
		if len(codeSlice) != 2 {
			global.Fatal("code param format is <ref>:</path/to/plugin.so>")
		}
		ref, err := insolar.NewReferenceFromString(codeSlice[0])
		if err != nil {
			global.Fatalf("Couldn't parse ref: %s", err.Error())
		}
		pluginPath := codeSlice[1]

		err = insider.AddPlugin(*ref, pluginPath)
		if err != nil {
			global.Fatalf("Couldn't add plugin by ref %s with .so from %s, err: %s ", ref, pluginPath, err.Error())
		}
	}

	err = rpc.Register(&ginsider.RPC{GI: insider})
	if err != nil {
		global.Fatal("Couldn't register RPC interface: ", err)
	}

	listener, err := net.Listen(*protocol, *listen)
	if err != nil {
		global.Fatal("couldn't setup listener on '"+*listen+"':", err)
	}

	var gracefulStop = make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	var waitChannel = make(chan bool)

	go func() {
		sig := <-gracefulStop
		global.Info("ginsider get signal: ", sig)
		close(waitChannel)
	}()

	if *metricsAddress != "" {
		ctx := context.Background() // TODO add tradeId and logger

		metricsConfiguration := configuration.Metrics{
			ListenAddress: *metricsAddress,
			Namespace:     "insgorund",
			ZpagesEnabled: true,
		}

		m := metrics.NewMetrics(metricsConfiguration, metrics.GetInsgorundRegistry(), "virtual")
		err = m.Start(ctx)
		if err != nil {
			global.Fatal("couldn't setup metrics ", err)
		}

		defer m.Stop(ctx) // nolint:errcheck
	}

	global.Debug("ginsider launched, listens " + *listen)
	go rpc.Accept(listener)

	<-waitChannel
	global.Debug("bye\n")
}
