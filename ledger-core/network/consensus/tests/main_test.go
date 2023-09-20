// +build never_run

package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
)

func TestConsensusMain(t *testing.T) {

	startedAt := time.Now()

	ctx := inslogger.WithLoggerLevel(context.Background(), log.DebugLevel)
	logger := inslogger.FromContext(ctx)
	global.SetLogger(logger)
	ctx = inslogger.SetLogger(ctx, global.Logger())
	_ = global.SetFilter(log.DebugLevel)

	netStrategy := NewDelayNetStrategy(DelayStrategyConf{
		MinDelay:         10 * time.Millisecond,
		MaxDelay:         30 * time.Millisecond,
		Variance:         0.2,
		SpikeProbability: 0.1,
	})
	strategyFactory := &EmuRoundStrategyFactory{}

	nodes := NewEmuNodeIntros(generateNameList(0, 1, 3, 5)...)
	netBuilder := newEmuNetworkBuilder(ctx, netStrategy, strategyFactory)

	for i := range nodes {
		netBuilder.connectEmuNode(nodes, i)
	}

	netBuilder.StartNetwork(ctx)

	netBuilder.StartPulsar(10, 2, "pulsar0", nodes)

	// time.AfterFunc(time.Second, func() {
	//	netBuilder.network.DropHost("V0007")
	// })

	for {
		fmt.Println("===", time.Since(startedAt), "=================================================")
		time.Sleep(time.Second)
		if time.Since(startedAt) > time.Minute*30 {
			return
		}
	}
}
