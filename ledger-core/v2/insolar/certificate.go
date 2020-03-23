// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insolar

import (
	"crypto"
)

type NodeMeta interface {
	GetNodeRef() *Reference
	GetPublicKey() crypto.PublicKey
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/insolar.Certificate -o ../testutils -s _mock.go -g

// Certificate interface provides methods to manage keys
type Certificate interface {
	AuthorizationCertificate

	GetRootDomainReference() *Reference
	GetDiscoveryNodes() []DiscoveryNode

	GetPulsarPublicKeys() []crypto.PublicKey

	GetMajorityRule() int
	GetMinRoles() (virtual uint, heavyMaterial uint, lightMaterial uint)
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/insolar.DiscoveryNode -o ../testutils -s _mock.go -g

type DiscoveryNode interface {
	NodeMeta

	GetRole() StaticRole
	GetHost() string
}

// AuthorizationCertificate interface provides methods to manage info about node from it certificate
type AuthorizationCertificate interface {
	NodeMeta

	GetRole() StaticRole
	SerializeNodePart() []byte
	GetDiscoverySigns() map[Reference][]byte
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/insolar.CertificateManager -o ../testutils -s _mock.go -g

// CertificateManager interface provides methods to manage nodes certificate
type CertificateManager interface {
	GetCertificate() Certificate
}
