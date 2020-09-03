// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniserver

import (
	"github.com/insolar/assured-ledger/ledger-core/crypto"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

type PeerCryptographyFactory interface {
	// TODO for some reason linter can't handle multiple declarations of the same method while it is valid for 1.14
	// cryptkit.DataSignatureVerifierFactory
	// cryptkit.DataSignerFactory
	CreateDataSignatureVerifier(cryptkit.SignatureKey) cryptkit.DataSignatureVerifier
	CreateDataSigner(cryptkit.SignatureKey) cryptkit.DataSigner
	IsSignatureKeySupported(cryptkit.SignatureKey) bool
	CreateDataDecrypter(cryptkit.SignatureKey) cryptkit.Decrypter
	CreateDataEncrypter(cryptkit.SignatureKey) cryptkit.Encrypter
	GetMaxSignatureSize() int
}

func NewPeerCryptographyFactory(scheme crypto.PlatformScheme) *peerCryptographyFactory {
	return &peerCryptographyFactory{
		scheme: scheme.TransportScheme(),
	}
}

type peerCryptographyFactory struct {
	scheme crypto.TransportScheme
}

func (v peerCryptographyFactory) CreateDataEncrypter(key cryptkit.SignatureKey) cryptkit.Encrypter {
	panic("implement me")
}

func (v peerCryptographyFactory) CreateDataDecrypter(cryptkit.SignatureKey) cryptkit.Decrypter {
	panic("implement me")
}

func (v peerCryptographyFactory) GetMaxSignatureSize() int {
	return 4
}

func (v peerCryptographyFactory) CreateDataSigner(k cryptkit.SignatureKey) cryptkit.DataSigner {
	// todo: how to get supported Signing Methods from scheme?
	// if k.GetSigningMethod() == testSigningMethod {
	// 	return TestDataSigner{}
	// }
	return cryptkit.AsDataSigner(v.scheme.PacketDigester(), v.scheme.PacketSigner())
}

func (v peerCryptographyFactory) IsSignatureKeySupported(k cryptkit.SignatureKey) bool {
	// todo: how to get supported Signing Methods ?
	return true // todo k.GetSigningMethod() == testSigningMethod
}

func (v peerCryptographyFactory) CreateDataSignatureVerifier(k cryptkit.SignatureKey) cryptkit.DataSignatureVerifier {
	// todo: how to get supported Signing Methods ?
	// if k.GetSigningMethod() == testSigningMethod {
	// 	return TestDataVerifier{}
	// }
	return cryptkit.AsDataSignatureVerifier(
		v.scheme.PacketDigester(),
		v.scheme.CreateSignatureVerifierWithPKS(nil),
		v.scheme.PacketSigner().GetSigningMethod())
}
