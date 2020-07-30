// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build slowtest,!longtest

package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	t.Skip("protocol version should not be exposed")
	t.Parallel()
	s := startNetworkSuite(t)
	defer s.stopNetworkSuite()

	testNode := s.newNetworkNode("testNode")
	s.preInitNode(testNode)
	s.InitNode(testNode)
	// testNode.serviceNetwork.NodeKeeper.GetOrigin().(node.MutableNode).SetVersion("ololo")
	// require.Equal(t, "ololo", testNode.serviceNetwork.NodeKeeper.GetOrigin().Version())
	err := testNode.componentManager.Start(s.ctx)
	assert.NoError(t, err)
	defer s.StopNode(testNode)

	assert.False(t, s.waitForNodeJoin(testNode.id, maxPulsesForJoin), "testNode joined with incorrect version")
}

func TestNodeLeave(t *testing.T) {
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

	// TODO actual implementation does NOT do graceful leave
	s := startNetworkSuite(t)
	defer s.stopNetworkSuite()

	testNode := s.startNewNetworkNode("testNode")
	assert.True(t, s.waitForNodeJoin(testNode.id, 3), "testNode not found in active list after 3 pulses")

	s.GracefulStop(testNode)

	assert.True(t, s.waitForNodeLeave(testNode.id, 3), "testNode found in active list after 3 pulses")

	s.AssertWorkingNodesCountDelta(0)
	s.AssertActiveNodesCountDelta(0)
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
