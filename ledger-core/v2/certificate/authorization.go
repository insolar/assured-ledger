// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package certificate

import (
	"crypto"

	"github.com/insolar/assured-ledger/ledger-core/v2/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/global"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"

	errors "github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

// AuthorizationCertificate holds info about node from it certificate
type AuthorizationCertificate struct {
	PublicKey      string                      `json:"public_key"`
	Reference      string                      `json:"reference"`
	Role           string                      `json:"role"`
	DiscoverySigns map[reference.Global][]byte `json:"-" codec:"discoverysigns"`

	nodePublicKey crypto.PublicKey
}

// GetPublicKey returns public key reference from node certificate
func (authCert *AuthorizationCertificate) GetPublicKey() crypto.PublicKey {
	return authCert.nodePublicKey
}

// GetNodeRef returns reference from node certificate
func (authCert *AuthorizationCertificate) GetNodeRef() reference.Global {
	ref, err := reference.GlobalFromString(authCert.Reference)
	if err != nil {
		global.Errorf("Invalid node reference in auth cert: %s\n", authCert.Reference)
		return reference.Global{}
	}
	return ref
}

// GetRole returns role from node certificate
func (authCert *AuthorizationCertificate) GetRole() node.StaticRole {
	return node.GetStaticRoleFromString(authCert.Role)
}

// GetDiscoverySigns return map of discovery nodes signs
func (authCert *AuthorizationCertificate) GetDiscoverySigns() map[reference.Global][]byte {
	return authCert.DiscoverySigns
}

// SerializeNodePart returns some node info decoded in bytes
func (authCert *AuthorizationCertificate) SerializeNodePart() []byte {
	return []byte(authCert.PublicKey + authCert.Reference + authCert.Role)
}

// SignNodePart signs node part in certificate
func (authCert *AuthorizationCertificate) SignNodePart(key crypto.PrivateKey) ([]byte, error) {
	signer := scheme.DataSigner(key, scheme.IntegrityHasher())
	sign, err := signer.Sign(authCert.SerializeNodePart())
	if err != nil {
		return nil, errors.W(err, "[ SignNodePart ] Can't Sign")
	}
	return sign.Bytes(), nil
}

// Deserialize deserializes data to AuthorizationCertificate interface
func Deserialize(data []byte, keyProc cryptography.KeyProcessor) (node.AuthorizationCertificate, error) {
	cert := &AuthorizationCertificate{}
	err := insolar.Deserialize(data, cert)

	if err != nil {
		return nil, errors.W(err, "[ AuthorizatonCertificate::Deserialize ] failed to deserialize a data")
	}

	key, err := keyProc.ImportPublicKeyPEM([]byte(cert.PublicKey))

	if err != nil {
		return nil, errors.W(err, "[ AuthorizationCertificate::Deserialize ] failed to import a public key")
	}

	cert.nodePublicKey = key

	return cert, nil
}

// Serialize serializes AuthorizationCertificate interface
func Serialize(authCert node.AuthorizationCertificate) ([]byte, error) {
	data, err := insolar.Serialize(authCert)
	if err != nil {
		return nil, errors.W(err, "[ AuthorizationCertificate::Serialize ]")
	}
	return data, nil
}
