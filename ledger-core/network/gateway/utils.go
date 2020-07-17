// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gateway

import (
	"bytes"
	"context"
	"crypto/rand"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/adapters"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/common/endpoints"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/serialization"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func GetBootstrapPulse(ctx context.Context, accessor beat.Accessor) (beat.Beat, error) {
	if pc, err := accessor.Latest(ctx); err == nil {
		return pc, nil
	}
	return beat.Beat{}, throw.E("latest pulse is not available")
}

func EnsureGetPulse(ctx context.Context, accessor beat.Accessor, pulseNumber pulse.Number) network.NetworkedPulse {
	pc, err := accessor.Of(ctx, pulseNumber)
	if err != nil {
		inslogger.FromContext(ctx).Panicf("Failed to fetch pulse: %d", pulseNumber)
	}
	return pc
}

func getAnnounceSignature(
	node nodeinfo.NetworkNode,
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

func (p *consensusProxy) ChangePulse(ctx context.Context, newPulse beat.Beat) {
	p.Gatewayer.Gateway().(adapters.PulseChanger).ChangePulse(ctx, newPulse)
}

func (p *consensusProxy) UpdateState(ctx context.Context, pulseNumber pulse.Number, nodes []nodeinfo.NetworkNode, cloudStateHash []byte) {
	p.Gatewayer.Gateway().(adapters.StateUpdater).UpdateState(ctx, pulseNumber, nodes, cloudStateHash)
}
