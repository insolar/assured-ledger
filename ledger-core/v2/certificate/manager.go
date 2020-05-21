// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package certificate

import (
	"crypto"

	errors "github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/v2/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
)

// CertificateManager is a component for working with current node certificate
type CertificateManager struct { // nolint:golint
	certificate node.Certificate
}

// NewCertificateManager returns new CertificateManager instance
func NewCertificateManager(cert node.Certificate) *CertificateManager {
	return &CertificateManager{certificate: cert}
}

// GetCertificate returns current node certificate
func (m *CertificateManager) GetCertificate() node.Certificate {
	return m.certificate
}

// VerifyAuthorizationCertificate verifies certificate from some node
func VerifyAuthorizationCertificate(cs cryptography.Service, discoveryNodes []node.DiscoveryNode, authCert node.AuthorizationCertificate) (bool, error) {
	if len(discoveryNodes) != len(authCert.GetDiscoverySigns()) {
		return false, nil
	}
	data := authCert.SerializeNodePart()
	for _, node := range discoveryNodes {
		sign := authCert.GetDiscoverySigns()[node.GetNodeRef()]
		ok := cs.Verify(node.GetPublicKey(), cryptography.SignatureFromBytes(sign), data)
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

// NewUnsignedCertificate creates new unsigned certificate by copying
func NewUnsignedCertificate(baseCert node.Certificate, pKey string, role string, ref string) (node.Certificate, error) {
	cert := baseCert.(*Certificate)
	newCert := Certificate{
		MajorityRule: cert.MajorityRule,
		MinRoles:     cert.MinRoles,
		AuthorizationCertificate: AuthorizationCertificate{
			PublicKey: pKey,
			Reference: ref,
			Role:      role,
		},
		PulsarPublicKeys: cert.PulsarPublicKeys,
		BootstrapNodes:   make([]BootstrapNode, len(cert.BootstrapNodes)),
	}
	for i, node := range cert.BootstrapNodes {
		newCert.BootstrapNodes[i].Host = node.Host
		newCert.BootstrapNodes[i].NodeRef = node.NodeRef
		newCert.BootstrapNodes[i].PublicKey = node.PublicKey
		newCert.BootstrapNodes[i].NetworkSign = node.NetworkSign
		newCert.BootstrapNodes[i].NodeRole = node.NodeRole
	}
	return &newCert, nil
}

// NewManagerReadCertificate constructor creates new CertificateManager component
func NewManagerReadCertificate(publicKey crypto.PublicKey, keyProcessor cryptography.KeyProcessor, certPath string) (*CertificateManager, error) {
	cert, err := ReadCertificate(publicKey, keyProcessor, certPath)
	if err != nil {
		return nil, errors.W(err, "[ NewManagerReadCertificate ] failed to read certificate:")
	}
	certManager := NewCertificateManager(cert)
	return certManager, nil
}
