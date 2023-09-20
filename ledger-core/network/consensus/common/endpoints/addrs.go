package endpoints

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/args"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

type Name string

func (addr Name) IsLocalHost() bool {
	return len(addr) == 0
}

func (addr Name) Equals(o Name) bool {
	return addr == o
}

func (addr Name) String() string {
	return string(addr)
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/network/consensus/common/endpoints.Outbound -o . -s _mock.go -g

type Outbound interface {
	GetEndpointType() NodeEndpointType
	GetRelayID() node.ShortNodeID
	GetNameAddress() Name
	GetIPAddress() IPAddress
	CanAccept(connection Inbound) bool
}

func EqualOutboundEndpoints(p, o Outbound) bool {
	if args.IsNil(p) || args.IsNil(o) {
		return false
	}
	if p == o {
		return true
	}

	if p.GetEndpointType() != o.GetEndpointType() {
		return false
	}
	switch p.GetEndpointType() {
	case NameEndpoint:
		return p.GetNameAddress() == o.GetNameAddress()
	case IPEndpoint:
		return p.GetIPAddress() == o.GetIPAddress()
	case RelayEndpoint:
		return p.GetRelayID() == o.GetRelayID()
	}
	panic("missing")
}

func EqualListOfOutboundEndpoints(p []Outbound, o []Outbound) bool {
	if len(p) != len(o) {
		return false
	}
	for i, pi := range p {
		if !EqualOutboundEndpoints(pi, o[i]) {
			return false
		}
	}
	return true
}

type NodeEndpointType uint8

const (
	IPEndpoint NodeEndpointType = iota
	NameEndpoint
	RelayEndpoint
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/network/consensus/common/endpoints.Inbound -o . -s _mock.go -g

type Inbound interface {
	GetNameAddress() Name
	//	GetIPAddress() packets.IPAddress // TODO
	GetTransportKey() cryptkit.SigningKeyHolder
	GetTransportCert() cryptkit.CertificateHolder
	AsByteString() longbits.ByteString
}

var _ Inbound = &InboundConnection{}

func NewHostIdentityFromHolder(h Inbound) InboundConnection {
	return InboundConnection{
		Addr: h.GetNameAddress(),
		Key:  h.GetTransportKey(),
		Cert: h.GetTransportCert(),
	}
}

type InboundConnection struct {
	Addr Name
	Key  cryptkit.SigningKeyHolder
	Cert cryptkit.CertificateHolder
}

func ShortNodeIDAsByteString(nodeID node.ShortNodeID) string {
	return fmt.Sprintf("node:%s",
		string([]byte{byte(nodeID), byte(nodeID >> 8), byte(nodeID >> 16), byte(nodeID >> 24)}))
}

func (v InboundConnection) String() string {
	return fmt.Sprintf("name:%s", v.Addr)
}

func (v *InboundConnection) AsByteString() longbits.ByteString {
	return longbits.ByteString(v.String())
}

func (v *InboundConnection) GetNameAddress() Name {
	return v.Addr
}

func (v *InboundConnection) GetTransportKey() cryptkit.SigningKeyHolder {
	return v.Key
}

func (v *InboundConnection) GetTransportCert() cryptkit.CertificateHolder {
	return v.Cert
}
