// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package tests

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/network/consensus"
)

const (
	defaultPulseDelta     = 2
	defaultTestDuration   = defaultPulseDelta * time.Second * 10
	defaultStartCaseAfter = 1 * time.Second
)

var strategy = NewDelayNetStrategy(DelayStrategyConf{
	MinDelay:         10 * time.Millisecond,
	MaxDelay:         30 * time.Millisecond,
	Variance:         0.2,
	SpikeProbability: 0.1,
})

func TestConsensusJoin(t *testing.T) {
	ctx := initLogger(t)

	nodes, err := generateNodes(0, 1, 3, 5, nil)
	require.NoError(t, err)

	joiners, err := generateNodes(0, 0, 6, 1, nodes.discoveryNodes)
	require.NoError(t, err)

	ns, err := initNodes(ctx, consensus.ReadyNetwork, nodes, strategy)
	require.NoError(t, err)

	js, err := initNodes(ctx, consensus.Joiner, joiners, strategy)
	require.NoError(t, err)

	initPulsar(ctx, defaultPulseDelta, ns)

	testCase(defaultTestDuration, defaultStartCaseAfter, func() {
		for i, joiner := range js.staticProfiles {
			err := ns.controllers[i].AddJoinCandidate(candidate{
				joiner,
				joiner.GetExtension(),
			})

			require.NoError(t, err)
		}
	})

	// require.Len(t, ns.nodeKeepers[0].GetNodeSnapshot().GetOnlineNodes(), len(nodes.nodes)+len(joiners.nodes))
}

func TestConsensusLeave(t *testing.T) {
	ctx := initLogger(t)

	nodes, err := generateNodes(0, 1, 3, 5, nil)
	require.NoError(t, err)

	ns, err := initNodes(ctx, consensus.ReadyNetwork, nodes, strategy)
	require.NoError(t, err)

	initPulsar(ctx, defaultPulseDelta, ns)

	testCase(defaultTestDuration, defaultStartCaseAfter, func() {
		nodeIdx := 1

		<-ns.controllers[nodeIdx].Leave(0)
		err := ns.transports[nodeIdx].Stop(ns.contexts[nodeIdx])
		require.NoError(t, err)
		ns.controllers[nodeIdx].Abort()
	})

	// require.Len(t, ns.nodeKeepers[0].GetNodeSnapshot().GetOnlineNodes(), len(nodes.nodes)-1)
}

func TestConsensusDrop(t *testing.T) {
	ctx := initLogger(t)

	nodes, err := generateNodes(0, 1, 3, 5, nil)
	require.NoError(t, err)

	ns, err := initNodes(ctx, consensus.ReadyNetwork, nodes, strategy)
	require.NoError(t, err)

	initPulsar(ctx, defaultPulseDelta, ns)

	testCase(defaultTestDuration, defaultStartCaseAfter, func() {
		nodeIdx := 1

		err := ns.transports[nodeIdx].Stop(ns.contexts[nodeIdx])
		require.NoError(t, err)
	})

	// require.Len(t, ns.nodeKeepers[0].GetNodeSnapshot().GetOnlineNodes(), len(nodes.nodes)-1)
}

func TestConsensusJoinLeave(t *testing.T) {
	t.Skip("Until phase 4 ready")

	ctx := initLogger(t)

	nodes, err := generateNodes(0, 1, 3, 5, nil)
	require.NoError(t, err)

	joiners, err := generateNodes(0, 0, 0, 1, nodes.discoveryNodes)
	require.NoError(t, err)

	ns, err := initNodes(ctx, consensus.ReadyNetwork, nodes, strategy)
	require.NoError(t, err)

	js, err := initNodes(ctx, consensus.Joiner, joiners, strategy)
	require.NoError(t, err)

	initPulsar(ctx, defaultPulseDelta, ns)

	testCase(defaultTestDuration, defaultStartCaseAfter, func() {
		wg := &sync.WaitGroup{}
		wg.Add(2)

		go func() {
			nodeIdx := len(joiners.nodes) + 1

			<-ns.controllers[nodeIdx].Leave(0)
			err := ns.transports[nodeIdx].Stop(ns.contexts[nodeIdx])
			assert.NoError(t, err)
			ns.controllers[nodeIdx].Abort()

			wg.Done()
		}()

		go func() {
			for i, joiner := range js.staticProfiles {
				err := ns.controllers[i].AddJoinCandidate(candidate{
					joiner,
					joiner.GetExtension(),
				})

				require.NoError(t, err)
			}

			wg.Done()
		}()

		wg.Wait()
	})

	// require.Len(t, ns.nodeKeepers[0].GetNodeSnapshot().GetOnlineNodes(), len(nodes.nodes)+len(joiners.nodes)-1)
}

func TestConsensusJoinDrop(t *testing.T) {
	t.Skip("Until phase 4 ready")

	ctx := initLogger(t)

	nodes, err := generateNodes(0, 1, 3, 5, nil)
	require.NoError(t, err)

	joiners, err := generateNodes(0, 0, 0, 1, nodes.discoveryNodes)
	require.NoError(t, err)

	ns, err := initNodes(ctx, consensus.ReadyNetwork, nodes, strategy)
	require.NoError(t, err)

	js, err := initNodes(ctx, consensus.Joiner, joiners, strategy)
	require.NoError(t, err)

	initPulsar(ctx, defaultPulseDelta, ns)

	testCase(defaultTestDuration, defaultStartCaseAfter, func() {
		wg := &sync.WaitGroup{}
		wg.Add(2)

		go func() {
			nodeIdx := len(joiners.nodes) + 1

			err := ns.transports[nodeIdx].Stop(ns.contexts[nodeIdx])
			assert.NoError(t, err)

			wg.Done()
		}()

		go func() {
			for i, joiner := range js.staticProfiles {
				err := ns.controllers[i].AddJoinCandidate(candidate{
					joiner,
					joiner.GetExtension(),
				})

				require.NoError(t, err)
			}

			wg.Done()
		}()

		wg.Wait()
	})

	// require.Len(t, ns.nodeKeepers[0].GetNodeSnapshot().GetOnlineNodes(), len(nodes.nodes)+len(joiners.nodes)-1)
}

func TestConsensusDropLeave(t *testing.T) {
	ctx := initLogger(t)

	nodes, err := generateNodes(0, 1, 3, 5, nil)
	require.NoError(t, err)

	ns, err := initNodes(ctx, consensus.ReadyNetwork, nodes, strategy)
	require.NoError(t, err)

	initPulsar(ctx, defaultPulseDelta, ns)

	testCase(defaultTestDuration, defaultStartCaseAfter, func() {
		wg := &sync.WaitGroup{}
		wg.Add(2)

		go func() {
			nodeIdx := 6

			<-ns.controllers[nodeIdx].Leave(0)
			err := ns.transports[nodeIdx].Stop(ns.contexts[nodeIdx])
			assert.NoError(t, err)
			ns.controllers[nodeIdx].Abort()

			wg.Done()
		}()

		go func() {
			nodeIdx := 7

			err := ns.transports[nodeIdx].Stop(ns.contexts[nodeIdx])
			assert.NoError(t, err)

			wg.Done()
		}()

		wg.Wait()
	})

	// require.Len(t, ns.nodeKeepers[0].GetNodeSnapshot().GetOnlineNodes(), len(nodes.nodes)-2)
}

func TestConsensusAll(t *testing.T) {
	t.Skip("Until phase 4 ready")

	ctx := initLogger(t)

	nodes, err := generateNodes(0, 1, 3, 5, nil)
	require.NoError(t, err)

	joiners, err := generateNodes(0, 0, 1, 1, nodes.discoveryNodes)
	require.NoError(t, err)

	ns, err := initNodes(ctx, consensus.ReadyNetwork, nodes, strategy)
	require.NoError(t, err)

	js, err := initNodes(ctx, consensus.Joiner, joiners, strategy)
	require.NoError(t, err)

	initPulsar(ctx, defaultPulseDelta, ns)

	testCase(defaultTestDuration, defaultStartCaseAfter, func() {
		wg := &sync.WaitGroup{}
		wg.Add(3)

		go func() {
			nodeIdx := len(joiners.nodes) + 1

			<-ns.controllers[nodeIdx].Leave(0)
			err := ns.transports[nodeIdx].Stop(ns.contexts[nodeIdx])
			assert.NoError(t, err)
			ns.controllers[nodeIdx].Abort()

			wg.Done()
		}()

		go func() {
			nodeIdx := len(joiners.nodes) + 2

			err := ns.transports[nodeIdx].Stop(ns.contexts[nodeIdx])
			assert.NoError(t, err)

			wg.Done()
		}()

		go func() {
			for i, joiner := range js.staticProfiles {
				err := ns.controllers[i].AddJoinCandidate(candidate{
					joiner,
					joiner.GetExtension(),
				})

				require.NoError(t, err)
			}

			wg.Done()
		}()

		wg.Wait()
	})

	// require.Len(t, ns.nodeKeepers[0].GetNodeSnapshot().GetOnlineNodes(), len(nodes.nodes)+len(joiners.nodes)-2)
}
