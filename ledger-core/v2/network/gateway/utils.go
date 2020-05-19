// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gateway

import (
	"bytes"
	"context"
	"crypto/rand"

	"github.com/insolar/assured-ledger/ledger-core/v2/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/v2/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/network"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/adapters"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/common/endpoints"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/serialization"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/storage"
	pulse2 "github.com/insolar/assured-ledger/ledger-core/v2/pulse"
)

func GetBootstrapPulse(ctx context.Context, accessor storage.PulseAccessor) pulse.Pulse {
	puls, err := accessor.GetLatestPulse(ctx)
	if err != nil {
		puls = *pulse.EphemeralPulse
	}

	return puls
}

func EnsureGetPulse(ctx context.Context, accessor storage.PulseAccessor, pulseNumber pulse2.Number) pulse.Pulse {
	pulse, err := accessor.GetPulse(ctx, pulseNumber)
	if err != nil {
		inslogger.FromContext(ctx).Panicf("Failed to fetch pulse: %d", pulseNumber)
	}

	return pulse
}

func getAnnounceSignature(
	node node.NetworkNode,
	isDiscovery bool,
	kp cryptography.KeyProcessor,
	keystore cryptography.KeyStore,
	scheme cryptography.PlatformCryptographyScheme,
) ([]byte, *cryptography.Signature, error) {

	brief := serialization.NodeBriefIntro{}
	brief.ShortID = node.ShortID()
	brief.SetPrimaryRole(adapters.StaticRoleToPrimaryRole(node.Role()))
	if isDiscovery {
		brief.SpecialRoles = member.SpecialRoleDiscovery
	}
	brief.StartPower = 10

	addr, err := endpoints.NewIPAddress(node.Address())
	if err != nil {
		return nil, nil, err
	}
	copy(brief.Endpoint[:], addr[:])

	pk, err := kp.ExportPublicKeyBinary(node.PublicKey())
	if err != nil {
		return nil, nil, err
	}

	copy(brief.NodePK[:], pk)

	buf := &bytes.Buffer{}
	err = brief.SerializeTo(nil, buf)
	if err != nil {
		return nil, nil, err
	}

	data := buf.Bytes()
	data = data[:len(data)-64]

	key, err := keystore.GetPrivateKey("")
	if err != nil {
		return nil, nil, err
	}

	digest := scheme.IntegrityHasher().Hash(data)
	sign, err := scheme.DigestSigner(key).Sign(digest)
	if err != nil {
		return nil, nil, err
	}

	return digest, sign, nil
}

func getKeyStore(cryptographyService cryptography.Service) cryptography.KeyStore {
	// TODO: hacked
	return cryptographyService.(*platformpolicy.NodeCryptographyService).KeyStore
}

type consensusProxy struct {
	Gatewayer network.Gatewayer
}

func (p consensusProxy) State() []byte {
	nshBytes := make([]byte, 64)
	_, _ = rand.Read(nshBytes)
	return nshBytes
}

func (p *consensusProxy) ChangePulse(ctx context.Context, newPulse pulse.Pulse) {
	p.Gatewayer.Gateway().(adapters.PulseChanger).ChangePulse(ctx, newPulse)
}

func (p *consensusProxy) UpdateState(ctx context.Context, pulseNumber pulse2.Number, nodes []node.NetworkNode, cloudStateHash []byte) {
	p.Gatewayer.Gateway().(adapters.StateUpdater).UpdateState(ctx, pulseNumber, nodes, cloudStateHash)
}
