// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package adapters

import (
	"github.com/insolar/assured-ledger/ledger-core/crypto/legacyadapter"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"

	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/transport"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/core"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/phasebundle"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

type ECDSASignatureVerifierFactory struct {
	digester legacyadapter.Sha3Digester512
	scheme   cryptography.PlatformCryptographyScheme
}

func NewECDSASignatureVerifierFactory(
	digester legacyadapter.Sha3Digester512,
	scheme cryptography.PlatformCryptographyScheme,
) *ECDSASignatureVerifierFactory {
	return &ECDSASignatureVerifierFactory{
		digester: digester,
		scheme:   scheme,
	}
}

func (vf *ECDSASignatureVerifierFactory) CreateSignatureVerifierWithPKS(pks cryptkit.PublicKeyStore) cryptkit.SignatureVerifier {
	return legacyadapter.NewECDSASignatureVerifier(vf.scheme, pks)
}

type TransportCryptographyFactory struct {
	verifierFactory *ECDSASignatureVerifierFactory
	digestFactory   *ConsensusDigestFactory
	scheme          cryptography.PlatformCryptographyScheme
}

func NewTransportCryptographyFactory(scheme cryptography.PlatformCryptographyScheme) *TransportCryptographyFactory {
	return &TransportCryptographyFactory{
		verifierFactory: NewECDSASignatureVerifierFactory(
			legacyadapter.NewSha3Digester512(scheme),
			scheme,
		),
		digestFactory: NewConsensusDigestFactory(scheme),
		scheme:        scheme,
	}
}

func (cf *TransportCryptographyFactory) CreateSignatureVerifierWithPKS(pks cryptkit.PublicKeyStore) cryptkit.SignatureVerifier {
	return cf.verifierFactory.CreateSignatureVerifierWithPKS(pks)
}

func (cf *TransportCryptographyFactory) GetDigestFactory() transport.ConsensusDigestFactory {
	return cf.digestFactory
}

func (cf *TransportCryptographyFactory) CreateNodeSigner(sks cryptkit.SecretKeyStore) cryptkit.DigestSigner {
	return legacyadapter.NewECDSADigestSigner(sks, cf.scheme)
}

func (cf *TransportCryptographyFactory) CreatePublicKeyStore(skh cryptkit.SignatureKeyHolder) cryptkit.PublicKeyStore {
	return legacyadapter.NewECDSAPublicKeyStore(skh)
}

type RoundStrategyFactory struct {
	bundleFactory core.PhaseControllersBundleFactory
}

func NewRoundStrategyFactory() *RoundStrategyFactory {
	return &RoundStrategyFactory{
		bundleFactory: phasebundle.NewStandardBundleFactoryDefault(),
	}
}

func (rsf *RoundStrategyFactory) CreateRoundStrategy(online census.OnlinePopulation, config api.LocalNodeConfiguration) (core.RoundStrategy, core.PhaseControllersBundle) {
	rs := NewRoundStrategy(config)
	pcb := rsf.bundleFactory.CreateControllersBundle(online, config)
	return rs, pcb

}

type TransportFactory struct {
	cryptographyFactory transport.CryptographyAssistant
	packetBuilder       transport.PacketBuilder
	packetSender        transport.PacketSender
}

func NewTransportFactory(
	cryptographyFactory transport.CryptographyAssistant,
	packetBuilder transport.PacketBuilder,
	packetSender transport.PacketSender,
) *TransportFactory {
	return &TransportFactory{
		cryptographyFactory: cryptographyFactory,
		packetBuilder:       packetBuilder,
		packetSender:        packetSender,
	}
}

func (tf *TransportFactory) GetPacketSender() transport.PacketSender {
	return tf.packetSender
}

func (tf *TransportFactory) GetPacketBuilder(signer cryptkit.DigestSigner) transport.PacketBuilder {
	return tf.packetBuilder
}

func (tf *TransportFactory) GetCryptographyFactory() transport.CryptographyAssistant {
	return tf.cryptographyFactory
}

type keyStoreFactory struct {
	keyProcessor cryptography.KeyProcessor
}

func (p *keyStoreFactory) CreatePublicKeyStore(keyHolder cryptkit.SignatureKeyHolder) cryptkit.PublicKeyStore {
	pk, err := p.keyProcessor.ImportPublicKeyBinary(longbits.AsBytes(keyHolder))
	if err != nil {
		panic(err)
	}
	return legacyadapter.NewECDSAPublicKeyStoreFromPK(pk)
}

func NewNodeProfileFactory(keyProcessor cryptography.KeyProcessor) profiles.Factory {
	return profiles.NewSimpleProfileIntroFactory(&keyStoreFactory{keyProcessor})
}

type ConsensusDigestFactory struct {
	scheme cryptography.PlatformCryptographyScheme
}

func (cdf *ConsensusDigestFactory) CreatePairDigester() cryptkit.PairDigester {
	panic("implement me") // TODO implement CreatePairDigester
}

func NewConsensusDigestFactory(scheme cryptography.PlatformCryptographyScheme) *ConsensusDigestFactory {
	return &ConsensusDigestFactory{
		scheme: scheme,
	}
}

func (cdf *ConsensusDigestFactory) CreateDataDigester() cryptkit.DataDigester {
	return legacyadapter.NewSha3Digester512(cdf.scheme)
}

func (cdf *ConsensusDigestFactory) CreateSequenceDigester() cryptkit.SequenceDigester {
	return NewSequenceDigester(legacyadapter.NewSha3Digester512(cdf.scheme))
}

func (cdf *ConsensusDigestFactory) CreateForkingDigester() cryptkit.ForkingDigester {
	return NewSequenceDigester(legacyadapter.NewSha3Digester512(cdf.scheme))
}

func (cdf *ConsensusDigestFactory) CreateAnnouncementDigester() cryptkit.ForkingDigester {
	return NewSequenceDigester(legacyadapter.NewSha3Digester512(cdf.scheme))
}

func (cdf *ConsensusDigestFactory) CreateGlobulaStateDigester() transport.StateDigester {
	return NewStateDigester(
		NewSequenceDigester(legacyadapter.NewSha3Digester512(cdf.scheme)),
	)
}
