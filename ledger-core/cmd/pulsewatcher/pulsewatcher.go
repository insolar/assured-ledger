package main

import (
	"context"
	"log"
	"time"

	"github.com/insolar/insconfig"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/pulsewatcher"
)

const EnvPrefix = "pulsewatcher"

func main() {
	pCfg := configuration.NewPulseWatcherConfiguration()
	paramsCfg := insconfig.Params{
		EnvPrefix:        EnvPrefix,
		ConfigPathGetter: &insconfig.DefaultPathGetter{},
	}
	insConfigurator := insconfig.New(paramsCfg)
	err := insConfigurator.Load(&pCfg)
	if err != nil {
		global.Fatal("failed to load configuration from file: ", err.Error())
	}

	ctx, _ := inslogger.InitGlobalNodeLogger(context.Background(), pCfg.Log, "", "pulsewatcher")

	if len(pCfg.Nodes) == 0 {
		log.Fatal("couldn't find any nodes in config file")
	}
	if pCfg.Interval == 0 {
		pCfg.Interval = 100 * time.Millisecond
	}

	pulsewatcher.Run(ctx, pCfg)
}
