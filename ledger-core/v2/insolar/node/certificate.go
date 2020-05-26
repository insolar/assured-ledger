// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package node

import (
	"crypto"

	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

type Meta interface {
	GetNodeRef() reference.Global
	GetPublicKey() crypto.PublicKey
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/insolar/node.Certificate -o ../../testutils -s _mock.go -g

// Certificate interface provides methods to manage keys
type Certificate interface {
	AuthorizationCertificate

	GetDiscoveryNodes() []DiscoveryNode

	GetMajorityRule() int
	GetMinRoles() (virtual uint, heavyMaterial uint, lightMaterial uint)
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/insolar/node.DiscoveryNode -o ../../testutils -s _mock.go -g

type DiscoveryNode interface {
	Meta

	GetRole() StaticRole
	GetHost() string
}

// AuthorizationCertificate interface provides methods to manage info about node from it certificate
type AuthorizationCertificate interface {
	Meta

	GetRole() StaticRole
	SerializeNodePart() []byte
	GetDiscoverySigns() map[reference.Global][]byte
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/insolar/node.CertificateManager -o ../../testutils -s _mock.go -g

// CertificateManager interface provides methods to manage nodes certificate
type CertificateManager interface {
	GetCertificate() Certificate
}
