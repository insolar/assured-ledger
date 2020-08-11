// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gateway

import (
	"bytes"
	"context"
	"crypto/ecdsa"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/appctl/chorus"
	"github.com/insolar/assured-ledger/ledger-core/crypto/legacyadapter"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/adapters"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/common/endpoints"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/serialization"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

// EnsureGetPulse checks if NodeKeeper got list for pulseNumber
func EnsureGetPulse(ctx context.Context, report network.Report) pulse.Data {
	if report.PulseData.IsEmpty() {
		inslogger.FromContext(ctx).Panicf("EnsureGetPulse PulseData.IsEmpty: %d", report.PulseNumber)
	}

	if report.PulseData.PulseNumber != report.PulseNumber {
		inslogger.FromContext(ctx).Panicf("EnsureGetPulse report.PulseData.PulseNumber != report.PulseNumber: %d", report.PulseNumber)
	}
	return report.PulseData
}

func CalcAnnounceSignature(nodeID node.ShortNodeID, role member.PrimaryRole, addr endpoints.IPAddress, startPower member.Power, isDiscovery bool,
	pk []byte, keystore cryptography.KeyStore, scheme cryptography.PlatformCryptographyScheme,
) ([]byte, *cryptography.Signature, error) {

	brief := serialization.NodeBriefIntro{}
	brief.ShortID = nodeID
	brief.SetPrimaryRole(role)
	if isDiscovery {
		brief.SpecialRoles = member.SpecialRoleDiscovery
	}

	// TODO start power level is not passed properly - needs fix
	brief.StartPower = adapters.DefaultStartPower // startPower

	copy(brief.Endpoint[:], addr[:])
	copy(brief.NodePK[:], pk)

	buf := &bytes.Buffer{}
	if err := brief.SerializeTo(nil, buf); err != nil {
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

func CreateLocalNodeProfile(nk beat.NodeKeeper, cert nodeinfo.Certificate, address string,
	keyProcessor cryptography.KeyProcessor, svc cryptography.Service, scheme cryptography.PlatformCryptographyScheme,
) (*adapters.StaticProfile, error) {
	ref := cert.GetNodeRef()
	if !reference.Equal(ref, nk.GetLocalNodeReference()) {
		panic(throw.IllegalState())
	}

	role := cert.GetRole()
	publicKey := cert.GetPublicKey()

	endpointAddr, err := endpoints.NewIPAddress(address)
	if err != nil {
		return nil, err
	}

	pk, err := keyProcessor.ExportPublicKeyBinary(publicKey)
	if err != nil {
		return nil, err
	}

	// TODO add StartPower and PowerSet to Certificate
	startPower := adapters.DefaultStartPower

	id := node.GenerateShortID(ref)
	digest, signature, err := CalcAnnounceSignature(id, role, endpointAddr, startPower,
		network.OriginIsDiscovery(cert), pk, getKeyStore(svc), scheme,
	)
	if err != nil {
		return nil, err
	}

	dig := cryptkit.NewDigest(longbits.NewBits512FromBytes(digest), legacyadapter.SHA3Digest512)
	sig := cryptkit.NewSignature(longbits.NewBits512FromBytes(signature.Bytes()), dig.GetDigestMethod().SignedBy(legacyadapter.SECP256r1Sign))
	dsg := cryptkit.NewSignedDigest(dig, sig)

	specialRole := member.SpecialRoleNone
	// isJoiner := false
	if network.IsDiscovery(ref, cert) {
		specialRole |= member.SpecialRoleDiscovery
		// } else {
		// 	isJoiner = true
	}

	return adapters.NewStaticProfile(id, role, specialRole, startPower,
		adapters.NewStaticProfileExtension(id, ref, sig),
		adapters.NewOutboundIP(endpointAddr),
		legacyadapter.NewECDSAPublicKeyStoreFromPK(publicKey),
		legacyadapter.NewECDSASignatureKeyHolder(publicKey.(*ecdsa.PublicKey), keyProcessor),
		dsg,
	), nil

	// verifier := g.transportCrypt.CreateSignatureVerifierWithPKS(staticProfile.GetPublicKeyStore())
	// var anp censusimpl.NodeProfileSlot
	// if isJoiner {
	// 	anp = censusimpl.NewJoinerProfile(staticProfile, verifier)
	// } else {
	// 	anp = censusimpl.NewNodeProfile(0, staticProfile, verifier, staticProfile.GetStartPower())
	// }
	// newOrigin := nodeinfo.NewNetworkNode(&anp)
	// g.NodeKeeper.SetInitialSnapshot([]nodeinfo.NetworkNode{newOrigin})
	// g.localCandidate = adapters.NewCandidate(staticProfile, g.KeyProcessor)
	// return nil
}


func getKeyStore(cryptographyService cryptography.Service) cryptography.KeyStore {
	// TODO: hacked
	return cryptographyService.(*platformpolicy.NodeCryptographyService).KeyStore
}

type consensusProxy struct {
	Gatewayer network.Gatewayer
}

func (p consensusProxy) RequestNodeState(fn chorus.NodeStateFunc) {
	p.Gatewayer.Gateway().RequestNodeState(fn)
}

func (p consensusProxy) CancelNodeState() {
	p.Gatewayer.Gateway().CancelNodeState()
}

func (p consensusProxy) ChangeBeat(ctx context.Context, _ api.UpstreamReport, newPulse beat.Beat) {
	p.Gatewayer.Gateway().OnPulseFromConsensus(ctx, newPulse)
}

func (p consensusProxy) UpdateState(ctx context.Context, beat beat.Beat) {
	p.Gatewayer.Gateway().UpdateState(ctx, beat)
}
