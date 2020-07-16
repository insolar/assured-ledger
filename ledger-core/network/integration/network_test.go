// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build slowtest, !longtest

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	node2 "github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/node"
)

func TestNetworkConsensusManyTimes(t *testing.T) {
	t.Parallel()
	s := startNetworkSuite(t)
	defer s.stopNetworkSuite()

	s.waitForConsensus(5)
	s.AssertActiveNodesCountDelta(0)
}

func TestJoinerNodeConnect(t *testing.T) {
	t.Parallel()
	s := startNetworkSuite(t)
	defer s.stopNetworkSuite()

	joinerNode := s.startNewNetworkNode("JoinerNode")
	defer s.StopNode(joinerNode)

	assert.True(t, s.waitForNodeJoin(joinerNode.id, maxPulsesForJoin), "JoinerNode not found in active list after 3 pulses")

	s.AssertActiveNodesCountDelta(1)
}

func TestNodeConnectInvalidVersion(t *testing.T) {
	t.Parallel()
	s := startNetworkSuite(t)
	defer s.stopNetworkSuite()

	testNode := s.newNetworkNode("testNode")
	s.preInitNode(testNode)
	s.InitNode(testNode)
	testNode.serviceNetwork.NodeKeeper.GetOrigin().(node.MutableNode).SetVersion("ololo")
	require.Equal(t, "ololo", testNode.serviceNetwork.NodeKeeper.GetOrigin().Version())
	err := testNode.componentManager.Start(s.ctx)
	assert.NoError(t, err)
	defer s.StopNode(testNode)

	assert.False(t, s.waitForNodeJoin(testNode.id, maxPulsesForJoin), "testNode joined with incorrect version")
}

func TestNodeLeave(t *testing.T) {
	// t.Skip("FIXME - may cause os.Exit")
	s := startNetworkSuite(t)
	defer s.stopNetworkSuite()

	testNode := s.startNewNetworkNode("testNode")
	assert.True(t, s.waitForNodeJoin(testNode.id, 3), "testNode not found in active list after 3 pulses")

	s.AssertActiveNodesCountDelta(1)
	s.AssertWorkingNodesCountDelta(1)

	s.StopNode(testNode)

	assert.True(t, s.waitForNodeLeave(testNode.id, 3), "testNode found in active list after 3 pulses")

	s.AssertWorkingNodesCountDelta(0)
	s.AssertActiveNodesCountDelta(0)
}

func TestNodeGracefulLeave(t *testing.T) {
	//	t.Skip("FIXME node GracefulStop")
	s := startNetworkSuite(t)
	defer s.stopNetworkSuite()

	testNode := s.startNewNetworkNode("testNode")
	assert.True(t, s.waitForNodeJoin(testNode.id, 3), "testNode not found in active list after 3 pulses")

	s.GracefulStop(testNode)

	assert.True(t, s.waitForNodeLeave(testNode.id, 3), "testNode found in active list after 3 pulses")

	s.AssertWorkingNodesCountDelta(0)
	s.AssertActiveNodesCountDelta(0)
}

func TestNodeLeaveAtETA(t *testing.T) {
	t.Skip("FIXME")
	s := startNetworkSuite(t)
	defer s.stopNetworkSuite()

	testNode := s.newNetworkNode("testNode")
	s.preInitNode(testNode)

	s.InitNode(testNode)
	leaveApprovedCalls := 0
	testNode.terminationHandler.OnLeaveApprovedMock.Set(func(p context.Context) {
		leaveApprovedCalls++
		s.StopNode(testNode)
	})
	s.StartNode(testNode)
	defer func(s *consensusSuite) {
		s.StopNode(testNode)
	}(s)

	// wait for node will be added at active and working lists
	s.waitForConsensus(3)
	s.AssertActiveNodesCountDelta(1)
	s.AssertWorkingNodesCountDelta(1)

	pulse, err := s.bootstrapNodes[0].serviceNetwork.PulseAccessor.GetLatestPulse(s.ctx)
	require.NoError(t, err)

	// next pulse will be last for this node
	// leaving in 3 pulses
	pulseDelta := pulse.NextPulseNumber - pulse.PulseNumber
	testNode.serviceNetwork.Leave(s.ctx, pulse.PulseNumber+3*pulseDelta)

	// node still active and working
	s.waitForConsensus(1)
	s.AssertActiveNodesCountDelta(1)
	s.AssertWorkingNodesCountDelta(1)

	// now node leaves, but it's still in active list
	s.waitForConsensus(1)
	s.AssertActiveNodesCountDelta(1)
	s.AssertWorkingNodesCountDelta(0)
	require.Equal(t, 1, leaveApprovedCalls)

	s.waitForConsensus(1)
	s.AssertActiveNodesCountDelta(0)
	s.AssertWorkingNodesCountDelta(0)
}

func TestNodeComeAfterAnotherNodeSendLeaveETA(t *testing.T) {
	t.Skip("FIXME")
	s := startNetworkSuite(t)
	defer s.stopNetworkSuite()

	leavingNode := s.newNetworkNode("leavingNode")
	s.preInitNode(leavingNode)

	s.InitNode(leavingNode)
	s.StartNode(leavingNode)
	defer func(s *consensusSuite) {
		s.StopNode(leavingNode)
	}(s)

	// wait for node will be added at active list
	s.waitForConsensus(2)
	activeNodes := s.bootstrapNodes[0].serviceNetwork.NodeKeeper.GetAccessor(0).GetActiveNodes()
	require.Equal(t, s.getNodesCount()+1, len(activeNodes))

	pulse, err := s.bootstrapNodes[0].serviceNetwork.PulseAccessor.GetLatestPulse(s.ctx)
	require.NoError(t, err)

	// leaving in 3 pulses
	pulseDelta := pulse.NextPulseNumber - pulse.PulseNumber
	leavingNode.serviceNetwork.Leave(s.ctx, pulse.PulseNumber+3*pulseDelta)

	// wait for leavingNode will be marked as leaving
	s.waitForConsensus(1)

	newNode := s.newNetworkNode("testNode")
	s.preInitNode(newNode)

	s.InitNode(newNode)
	s.StartNode(newNode)
	defer func(s *consensusSuite) {
		s.StopNode(newNode)
	}(s)

	// wait for newNode will be added at active list, its a last pulse for leavingNode
	s.waitForConsensus(2)

	// newNode doesn't have workingNodes
	activeNodes = s.bootstrapNodes[0].GetActiveNodes()
	workingNodes := s.bootstrapNodes[0].GetWorkingNodes()
	newNodeWorkingNodes := newNode.GetWorkingNodes()

	require.Equal(t, s.getNodesCount()+2, len(activeNodes))
	require.Equal(t, s.getNodesCount()+1, len(workingNodes))
	require.Equal(t, 0, len(newNodeWorkingNodes))

	// newNode have to have same working node list as other nodes, but it doesn't because it miss leaving claim
	s.waitForConsensus(1)
	activeNodes = s.bootstrapNodes[0].GetActiveNodes()
	workingNodes = s.bootstrapNodes[0].GetWorkingNodes()
	newNodeWorkingNodes = newNode.GetWorkingNodes()

	require.Equal(t, s.getNodesCount()+2, len(activeNodes))
	require.Equal(t, s.getNodesCount()+1, len(workingNodes))
	// TODO: fix this testcase
	require.Equal(t, s.getNodesCount()+1, len(newNodeWorkingNodes))

	// leaveNode leaving, newNode still ok
	s.waitForConsensus(1)
	activeNodes = s.bootstrapNodes[0].GetActiveNodes()
	workingNodes = newNode.GetWorkingNodes()
	newNodeWorkingNodes = newNode.GetWorkingNodes()

	require.Equal(t, s.getNodesCount()+1, len(activeNodes))
	require.Equal(t, s.getNodesCount()+1, len(workingNodes))
	require.Equal(t, s.getNodesCount()+1, len(newNodeWorkingNodes))
}

func TestDiscoveryDown(t *testing.T) {
	t.Skip("FIXME - causes unhandled panic by AbortMock")
	s := startNetworkSuite(t)
	defer s.stopNetworkSuite()

	s.StopNode(s.bootstrapNodes[0])
	s.waitForConsensusExcept(2, s.bootstrapNodes[0].id)
	for i := 1; i < s.getNodesCount(); i++ {
		activeNodes := s.bootstrapNodes[i].GetWorkingNodes()
		require.Equal(t, s.getNodesCount()-1, len(activeNodes))
	}
}

func flushNodeKeeper(keeper network.NodeKeeper) {
	// keeper.SetIsBootstrapped(false)
	// keeper.SetCloudHash(nil)
	keeper.SetInitialSnapshot([]node2.NetworkNode{})
	keeper.GetOrigin().(node.MutableNode).SetState(node2.Ready)
}

func TestDiscoveryRestart(t *testing.T) {
	t.Skip("FIXME - fails with NPE at nodekeeper.MoveSyncToActive")
	s := startNetworkSuite(t)
	defer s.stopNetworkSuite()

	s.waitForConsensus(2)

	global.Info("Discovery node stopping...")
	err := s.bootstrapNodes[0].serviceNetwork.Stop(context.Background())
	flushNodeKeeper(s.bootstrapNodes[0].serviceNetwork.NodeKeeper)
	global.Info("Discovery node stopped...")
	require.NoError(t, err)

	s.waitForConsensusExcept(2, s.bootstrapNodes[0].id)
	activeNodes := s.bootstrapNodes[1].GetWorkingNodes()
	require.Equal(t, s.getNodesCount()-1, len(activeNodes))

	global.Info("Discovery node starting...")
	err = s.bootstrapNodes[0].serviceNetwork.Start(context.Background())
	global.Info("Discovery node started")
	require.NoError(t, err)

	s.waitForConsensusExcept(3, s.bootstrapNodes[0].id)
	activeNodes = s.bootstrapNodes[1].GetWorkingNodes()
	require.Equal(t, s.getNodesCount(), len(activeNodes))
	activeNodes = s.bootstrapNodes[0].GetWorkingNodes()
	require.Equal(t, s.getNodesCount(), len(activeNodes))
}

func TestDiscoveryRestartNoWait(t *testing.T) {
	t.Skip("FIXME - fails with NPE at nodekeeper.MoveSyncToActive")
	s := startNetworkSuite(t)
	defer s.stopNetworkSuite()

	s.waitForConsensus(2)

	global.Info("Discovery node stopping...")
	err := s.bootstrapNodes[0].serviceNetwork.Stop(context.Background())
	flushNodeKeeper(s.bootstrapNodes[0].serviceNetwork.NodeKeeper)
	global.Info("Discovery node stopped...")
	require.NoError(t, err)

	go func(s *consensusSuite) {
		global.Info("Discovery node starting...")
		err = s.bootstrapNodes[0].serviceNetwork.Start(context.Background())
		global.Info("Discovery node started")
		require.NoError(t, err)
	}(s)

	s.waitForConsensusExcept(4, s.bootstrapNodes[0].id)
	activeNodes := s.bootstrapNodes[1].GetActiveNodes()
	require.Equal(t, s.getNodesCount(), len(activeNodes))
	s.waitForConsensusExcept(1, s.bootstrapNodes[0].id)
	activeNodes = s.bootstrapNodes[0].GetWorkingNodes()
	require.Equal(t, s.getNodesCount(), len(activeNodes))
	activeNodes = s.bootstrapNodes[1].GetWorkingNodes()
	require.Equal(t, s.getNodesCount(), len(activeNodes))
}

func TestBootstrap(t *testing.T) {
	t.Skip("FIXME - occasional NPE")
	s := startBootstrapSuite(t)
	defer s.stopBootstrapSuite()

	s.StartNodesNetwork(s.bootstrapNodes)

	s.waitForConsensus(2)
	s.AssertActiveNodesCountDelta(0)

	s.waitForConsensus(1)
	s.AssertActiveNodesCountDelta(0)
	s.AssertWorkingNodesCountDelta(0)
}
