package nodeinfo

import (
	"context"
	"crypto"

	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type Meta interface {
	GetNodeRef() reference.Global
	GetPublicKey() crypto.PublicKey
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/network/nodeinfo.Certificate -o ../../testutils -s _mock.go -g

// Certificate interface provides methods to manage keys
type Certificate interface {
	AuthorizationCertificate

	GetDiscoveryNodes() []DiscoveryNode

	GetMajorityRule() int
	GetMinRoles() (virtual uint, heavyMaterial uint, lightMaterial uint)
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/network/nodeinfo.DiscoveryNode -o ../../testutils -s _mock.go -g

type DiscoveryNode interface {
	Meta
	GetHost() string
}

// AuthorizationCertificate interface provides methods to manage info about node from it certificate
type AuthorizationCertificate interface {
	Meta

	GetRole() member.PrimaryRole
	SerializeNodePart() []byte
	GetDiscoverySigns() map[reference.Global][]byte
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/network/nodeinfo.CertificateManager -o ../../testutils -s _mock.go -g

// CertificateManager interface provides methods to manage nodes certificate
type CertificateManager interface {
	GetCertificate() Certificate
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/network/nodeinfo.CertificateGetter -o ../../testutils -s _mock.go -g

type CertificateGetter interface {
	// GetCert registers reference and returns new certificate for it
	GetCert(context.Context, reference.Global) (Certificate, error)
}

