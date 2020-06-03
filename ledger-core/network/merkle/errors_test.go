// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package merkle

import (
	"context"
	"crypto"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	errors "github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"

	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/v2/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/v2/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/v2/network"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulsar/pulsartestutils"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils"
	network2 "github.com/insolar/assured-ledger/ledger-core/v2/testutils/network"
)

type calculatorErrorSuite struct {
	suite.Suite

	pulse          *pulsestor.Pulse
	originProvider network.OriginProvider
	service        cryptography.Service

	calculator Calculator
}

func (t *calculatorErrorSuite) TestGetNodeProofError() {
	ph, np, err := t.calculator.GetPulseProof(&PulseEntry{Pulse: t.pulse})

	t.Assert().Error(err)
	t.Assert().Contains(err.Error(), "[ GetPulseProof ] Failed to sign node info hash")
	t.Assert().Nil(np)
	t.Assert().Nil(ph)
}

func (t *calculatorErrorSuite) TestGetGlobuleProofCalculateError() {
	pulseEntry := &PulseEntry{Pulse: t.pulse}

	prevCloudHash, _ := hex.DecodeString(
		"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
	)

	globuleEntry := &GlobuleEntry{
		PulseEntry:    pulseEntry,
		PulseHash:     nil,
		ProofSet:      nil,
		PrevCloudHash: prevCloudHash,
		GlobuleID:     0,
	}
	gh, gp, err := t.calculator.GetGlobuleProof(globuleEntry)

	t.Assert().Error(err)
	t.Assert().Contains(err.Error(), "[ GetGlobuleProof ] Failed to calculate node root")
	t.Assert().Nil(gh)
	t.Assert().Nil(gp)
}

func (t *calculatorErrorSuite) TestGetGlobuleProofSignError() {
	pulseEntry := &PulseEntry{Pulse: t.pulse}

	prevCloudHash, _ := hex.DecodeString(
		"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
	)

	globuleEntry := &GlobuleEntry{
		PulseEntry: pulseEntry,
		PulseHash:  nil,
		ProofSet: map[node.NetworkNode]*PulseProof{
			t.originProvider.GetOrigin(): {},
		},
		PrevCloudHash: prevCloudHash,
		GlobuleID:     0,
	}
	gh, gp, err := t.calculator.GetGlobuleProof(globuleEntry)

	t.Assert().Error(err)
	t.Assert().Contains(err.Error(), "[ GetGlobuleProof ] Failed to sign globule hash")
	t.Assert().Nil(gh)
	t.Assert().Nil(gp)
}

func (t *calculatorErrorSuite) TestGetCloudProofSignError() {
	ch, cp, err := t.calculator.GetCloudProof(&CloudEntry{
		ProofSet: []*GlobuleProof{
			{},
		},
		PrevCloudHash: nil,
	})

	t.Assert().Error(err)
	t.Assert().Contains(err.Error(), "[ GetCloudProof ] Failed to sign cloud hash")
	t.Assert().Nil(ch)
	t.Assert().Nil(cp)
}

func (t *calculatorErrorSuite) TestGetCloudProofCalculateError() {
	ch, cp, err := t.calculator.GetCloudProof(&CloudEntry{
		ProofSet:      nil,
		PrevCloudHash: nil,
	})

	t.Assert().Error(err)
	t.Assert().Contains(err.Error(), "[ GetCloudProof ] Failed to calculate cloud hash")
	t.Assert().Nil(ch)
	t.Assert().Nil(cp)
}

func TestCalculatorError(t *testing.T) {
	calculator := &calculator{}

	cm := component.NewManager(nil)

	key, _ := platformpolicy.NewKeyProcessor().GeneratePrivateKey()
	require.NotNil(t, key)

	service := cryptography.NewServiceMock(t)
	service.SignMock.Set(func(p []byte) (r *cryptography.Signature, r1 error) {
		return nil, errors.New("Sign error")
	})
	service.GetPublicKeyMock.Set(func() (r crypto.PublicKey, r1 error) {
		return "key", nil
	})
	scheme := platformpolicy.NewPlatformCryptographyScheme()

	ps := pulsestor.NewStorageMem()

	op := network2.NewOriginProviderMock(t)
	op.GetOriginMock.Set(func() node.NetworkNode {
		return createOrigin()
	})

	th := testutils.NewTerminationHandlerMock(t)

	am := staterMock{
		stateFunc: func() []byte {
			return []byte{1, 2, 3}
		},
	}

	cm.Inject(th, op, &am, calculator, service, scheme, ps)

	require.NotNil(t, calculator.Stater)
	require.NotNil(t, calculator.OriginProvider)
	require.NotNil(t, calculator.CryptographyService)
	require.NotNil(t, calculator.PlatformCryptographyScheme)

	err := cm.Init(context.Background())
	require.NoError(t, err)

	pulseObject := &pulsestor.Pulse{
		PulseNumber:     1337,
		NextPulseNumber: 1347,
		Entropy:         pulsartestutils.MockEntropyGenerator{}.GenerateEntropy(),
	}

	s := &calculatorErrorSuite{
		Suite:          suite.Suite{},
		calculator:     calculator,
		pulse:          pulseObject,
		originProvider: op,
		service:        service,
	}
	suite.Run(t, s)
}

type staterMock struct {
	stateFunc func() []byte
}

func (m staterMock) State() []byte {
	return m.stateFunc()
}
