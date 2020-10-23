// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulsenetwork

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/keystore"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/common/endpoints"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/transport"
	"github.com/insolar/assured-ledger/ledger-core/pulsar"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

const (
	PULSENUMBER = pulse.MinTimePulse + 155
)

type pProcessor struct{}

func (pp *pProcessor) ProcessPacket(ctx context.Context, payload transport.PacketParser, from endpoints.Inbound) error {
	fmt.Printf("ProcessPacket from %s : %v", from.GetNameAddress(), payload)
	return nil
}

func TestDistributor_Distribute(t *testing.T) {
	instestlogger.SetTestOutput(t)

	nodeServ := NewPulsarUniserver()
	pulsarServ := NewPulsarUniserver()
	nodeServ.StartListen()
	pulsarServ.StartListen()

	ctx := context.Background()

	// handler := func(ctx context.Context, r network.ReceivedPacket) (network.Packet, error) {
	// 	global.Info("handle Pulse")
	// 	pulse := r.GetRequest().GetPulse()
	// 	assert.EqualValues(t, PULSENUMBER, pulse.Pulse.PulseNumber)
	// 	return nil, nil
	// }
	// n1.RegisterRequestHandler(types.Pulse, handler)

	// err = n1.Start(ctx)
	// require.NoError(t, err)
	// defer func() {
	// 	err = n1.Stop(ctx)
	// 	require.NoError(t, err)
	// }()

	address := nodeServ.PeerManager().Local().GetPrimary().String()

	pulsarCfg := configuration.NewPulsar()
	pulsarCfg.PulseDistributor.BootstrapHosts = []string{address}

	d, err := NewDistributor(pulsarCfg.PulseDistributor, pulsarServ)
	require.NoError(t, err)
	assert.NotNil(t, d)

	cm := component.NewManager(nil)
	cm.SetLogger(global.Logger())

	key, err := platformpolicy.NewKeyProcessor().GeneratePrivateKey()
	require.NoError(t, err)

	cm.Inject(d, platformpolicy.NewPlatformCryptographyScheme(), keystore.NewInplaceKeyStore(key))
	err = cm.Init(ctx)
	require.NoError(t, err)
	err = cm.Start(ctx)
	require.NoError(t, err)
	defer func() {
		err = cm.Stop(ctx)
		require.NoError(t, err)
	}()

	d.Distribute(ctx, pulsar.PulsePacket{PulseNumber: PULSENUMBER})
	d.Distribute(ctx, pulsar.PulsePacket{PulseNumber: PULSENUMBER})
	d.Distribute(ctx, pulsar.PulsePacket{PulseNumber: PULSENUMBER})
}
