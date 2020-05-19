// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulsar

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/configuration"
	"github.com/insolar/assured-ledger/ledger-core/v2/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/v2/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulsar/entropygenerator"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils"
)

func TestPulsar_Send(t *testing.T) {
	distMock := testutils.NewPulseDistributorMock(t)
	var pn pulse.Number = pulse.MinTimePulse

	distMock.DistributeMock.Set(func(ctx context.Context, p1 insolar.Pulse) {
		require.Equal(t, pn, p1.PulseNumber)
		require.NotNil(t, p1.Entropy)
	})

	pcs := platformpolicy.NewPlatformCryptographyScheme()
	crypto := cryptography.NewServiceMock(t)
	crypto.SignMock.Return(&cryptography.Signature{}, nil)
	proc := platformpolicy.NewKeyProcessor()
	key, err := proc.GeneratePrivateKey()
	require.NoError(t, err)
	crypto.GetPublicKeyMock.Return(proc.ExtractPublicKey(key), nil)

	p := NewPulsar(
		configuration.NewPulsar(),
		crypto,
		pcs,
		platformpolicy.NewKeyProcessor(),
		distMock,
		&entropygenerator.StandardEntropyGenerator{},
	)

	err = p.Send(context.TODO(), pn)

	require.NoError(t, err)
	require.Equal(t, pn, p.LastPN())

	distMock.MinimockWait(1 * time.Minute)
}
