// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"fmt"
	"strconv"

	"github.com/insolar/consensus-reports/pkg/middleware"
)

type ConsensusTestCallbacks struct{}

func NewConsensusTestCallbacks() *ConsensusTestCallbacks {
	return &ConsensusTestCallbacks{}
}

func (c *ConsensusTestCallbacks) started(params NetParams, timing eventTiming) {
}

func (c *ConsensusTestCallbacks) ready(params NetParams, timing eventTiming) {
	fmt.Printf("Network is ready at %s (%d) with %d nodes. Running consensus for %s\n",
		timing.stoppedAt.Format("15:04:05"),
		timing.stoppedAt.Unix(),
		params.NodesCount,
		params.WaitInReadyState.String(),
	)

	newRange := middleware.RangeConfig{
		StartTime: timing.stoppedAt.Unix(),
		Interval:  params.WaitInReadyState,
	}
	newRange.Properties = append(newRange.Properties, middleware.PropertyConfig{
		Name:  "network_size",
		Value: strconv.Itoa(int(params.NodesCount)),
	})
	Groups[0].Ranges = append(Groups[0].Ranges, newRange)
}

func (c *ConsensusTestCallbacks) stopped(params NetParams, timing eventTiming) {
}

func (c *ConsensusTestCallbacks) suiteFinished(config *KubeDeployToolConfig) error {
	fmt.Println("Gathering metrics")
	dirName, err := CollectMetrics(config.MetricParams, Groups)
	if err != nil {
		return fmt.Errorf("metrics replicator failed: %s", err)
	}
	fmt.Println("Done")

	fmt.Println("Creating report")
	reportURL, err := CreateReport(config.MetricParams, dirName)
	if err != nil {
		return fmt.Errorf("failed to create report: %s", err)
	}
	fmt.Printf("Done\nReport is available at %s\n", reportURL)
	return nil
}
