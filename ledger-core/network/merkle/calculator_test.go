// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package merkle

import (
	"context"
	"encoding/hex"
	"testing"

	node2 "github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/node"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	network2 "github.com/insolar/assured-ledger/ledger-core/testutils/network"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/pulsar/pulsartestutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
)

func createOrigin() node2.NetworkNode {
	ref, _ := reference.GlobalFromString("insolar:1MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI")
	return node.NewNode(ref, node2.StaticRoleVirtual, nil, "127.0.0.1:5432", "")
}

type calculatorSuite struct {
	suite.Suite

	pulse          *pulsestor.Pulse
	originProvider network.OriginProvider
	service        cryptography.Service

	calculator Calculator
}

func (t *calculatorSuite) TestGetNodeProof() {
	ph, np, err := t.calculator.GetPulseProof(&PulseEntry{Pulse: t.pulse})

	t.Assert().NoError(err)
	t.Assert().NotNil(np)

	key, err := t.service.GetPublicKey()
	t.Assert().NoError(err)

	t.Assert().True(t.calculator.IsValid(np, ph, key))
}

func (t *calculatorSuite) TestGetGlobuleProof() {
	pulseEntry := &PulseEntry{Pulse: t.pulse}
	ph, pp, err := t.calculator.GetPulseProof(pulseEntry)
	t.Assert().NoError(err)

	prevCloudHash, _ := hex.DecodeString(
		"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
	)

	globuleEntry := &GlobuleEntry{
		PulseEntry: pulseEntry,
		PulseHash:  ph,
		ProofSet: map[node2.NetworkNode]*PulseProof{
			t.originProvider.GetOrigin(): pp,
		},
		PrevCloudHash: prevCloudHash,
		GlobuleID:     0,
	}
	gh, gp, err := t.calculator.GetGlobuleProof(globuleEntry)

	t.Assert().NoError(err)
	t.Assert().NotNil(gp)

	key, err := t.service.GetPublicKey()
	t.Assert().NoError(err)

	valid := t.calculator.IsValid(gp, gh, key)
	t.Assert().True(valid)
}

func (t *calculatorSuite) TestGetCloudProof() {
	pulseEntry := &PulseEntry{Pulse: t.pulse}
	ph, pp, err := t.calculator.GetPulseProof(pulseEntry)
	t.Assert().NoError(err)

	prevCloudHash, _ := hex.DecodeString(
		"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
	)

	globuleEntry := &GlobuleEntry{
		PulseEntry: pulseEntry,
		PulseHash:  ph,
		ProofSet: map[node2.NetworkNode]*PulseProof{
			t.originProvider.GetOrigin(): pp,
		},
		PrevCloudHash: prevCloudHash,
		GlobuleID:     0,
	}
	_, gp, err := t.calculator.GetGlobuleProof(globuleEntry)

	ch, cp, err := t.calculator.GetCloudProof(&CloudEntry{
		ProofSet:      []*GlobuleProof{gp},
		PrevCloudHash: prevCloudHash,
	})

	t.Assert().NoError(err)
	t.Assert().NotNil(gp)

	key, err := t.service.GetPublicKey()
	t.Assert().NoError(err)

	valid := t.calculator.IsValid(cp, ch, key)
	t.Assert().True(valid)
}

func TestNewCalculator(t *testing.T) {
	c := NewCalculator()
	require.NotNil(t, c)
}

func TestCalculator(t *testing.T) {
	calculator := &calculator{}

	key, _ := platformpolicy.NewKeyProcessor().GeneratePrivateKey()
	require.NotNil(t, key)

	service := platformpolicy.NewKeyBoundCryptographyService(key)
	scheme := platformpolicy.NewPlatformCryptographyScheme()
	op := network2.NewOriginProviderMock(t)
	op.GetOriginMock.Set(func() node2.NetworkNode {
		return createOrigin()
	})

	th := testutils.NewTerminationHandlerMock(t)
	am := staterMock{
		stateFunc: func() []byte {
			return []byte("state")
		},
	}

	cm := component.NewManager(nil)
	cm.SetLogger(global.Logger())
	cm.Inject(th, op, &am, calculator, service, scheme)

	require.NotNil(t, calculator.Stater)
	require.NotNil(t, calculator.OriginProvider)
	require.NotNil(t, calculator.CryptographyService)
	require.NotNil(t, calculator.PlatformCryptographyScheme)

	err := cm.Init(context.Background())
	require.NoError(t, err)

	pulse := &pulsestor.Pulse{
		PulseNumber:     1337,
		NextPulseNumber: 1347,
		Entropy:         pulsartestutils.MockEntropyGenerator{}.GenerateEntropy(),
	}

	s := &calculatorSuite{
		Suite:          suite.Suite{},
		calculator:     calculator,
		pulse:          pulse,
		originProvider: op,
		service:        service,
	}
	suite.Run(t, s)
}
