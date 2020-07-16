// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nodeinfo

import (
	"crypto"

	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type Meta interface {
	GetNodeRef() reference.Global
	GetPublicKey() crypto.PublicKey
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/insolar/node.Certificate -o ../../testutils -s _mock.go -g

// Certificate interface provides methods to manage keys
type Certificate interface {
	AuthorizationCertificate

	GetDiscoveryNodes() []DiscoveryNode

	GetMajorityRule() int
	GetMinRoles() (virtual uint, heavyMaterial uint, lightMaterial uint)
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/insolar/node.DiscoveryNode -o ../../testutils -s _mock.go -g

type DiscoveryNode interface {
	Meta

	GetRole() member.PrimaryRole
	GetHost() string
}

// AuthorizationCertificate interface provides methods to manage info about node from it certificate
type AuthorizationCertificate interface {
	Meta

	GetRole() member.PrimaryRole
	SerializeNodePart() []byte
	GetDiscoverySigns() map[reference.Global][]byte
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/insolar/node.CertificateManager -o ../../testutils -s _mock.go -g

// CertificateManager interface provides methods to manage nodes certificate
type CertificateManager interface {
	GetCertificate() Certificate
}
