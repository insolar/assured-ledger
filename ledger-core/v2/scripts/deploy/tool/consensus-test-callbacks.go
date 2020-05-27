// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"fmt"
)

type ConsensusTestCallbacks struct{}

func NewConsensusTestCallbacks() *ConsensusTestCallbacks {
	return &ConsensusTestCallbacks{}
}

func (c *ConsensusTestCallbacks) started(params NetParams, timing eventTiming) {
	fmt.Println("callback started called")
	fmt.Println("timing: %w", timing)
	fmt.Println("params: %w", params)
}

func (c *ConsensusTestCallbacks) ready(params NetParams, timing eventTiming) {
	fmt.Println("callback ready called")
	fmt.Println("timing: %w", timing)
	fmt.Println("params: %w", params)
}

func (c *ConsensusTestCallbacks) stopped(params NetParams, timing eventTiming) {
	fmt.Println("callback stopped called")
	fmt.Println("timing: %w", timing)
	fmt.Println("params: %w", params)
}
