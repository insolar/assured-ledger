package integration

import (
	"fmt"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type bootstrapSuite struct {
	testSuite
}

func (s *bootstrapSuite) Setup() {
	var err error
	s.pulsar, err = NewTestPulsar(reqTimeoutMs*10, pulseDelta*10)
	require.NoError(s.t, err)

	inslogger.FromContext(s.ctx).Info("SetupTest")

	for i := 0; i < s.bootstrapCount; i++ {
		role := member.PrimaryRoleVirtual
		if i == 0 {
			role = member.PrimaryRoleHeavyMaterial
		}

		s.bootstrapNodes = append(s.bootstrapNodes, s.newNetworkNodeWithRole(fmt.Sprintf("bootstrap_%d", i), role))
	}

	s.SetupNodesNetwork(s.bootstrapNodes)

	pulseReceivers := make([]string, 0)
	for _, node := range s.bootstrapNodes {
		pulseReceivers = append(pulseReceivers, node.host)
	}

	global.Info("Start test pulsar")
	err = s.pulsar.Start(s.ctx, pulseReceivers)
	require.NoError(s.t, err)
}

func (s *bootstrapSuite) stopBootstrapSuite() {
	logger := inslogger.FromContext(s.ctx)
	logger.Info("stopNetworkSuite")

	logger.Info("Stop bootstrap nodes")
	for _, n := range s.bootstrapNodes {
		err := n.componentManager.Stop(n.ctx)
		assert.NoError(s.t, err)
	}
}

func (s *bootstrapSuite) waitForConsensus(consensusCount int) {
	for i := 0; i < consensusCount; i++ {
		for _, n := range s.bootstrapNodes {
			<-n.consensusResult
		}
	}
}

func newBootstraptSuite(t *testing.T, bootstrapCount int) *bootstrapSuite {
	return &bootstrapSuite{
		testSuite: newTestSuite(t, bootstrapCount, 0),
	}
}

func startBootstrapSuite(t *testing.T) *bootstrapSuite {
	s := newBootstraptSuite(t, 11)
	s.Setup()
	return s
}
