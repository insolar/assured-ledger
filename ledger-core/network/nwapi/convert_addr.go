package nwapi

import (
	"net"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func FromUDPAddr(a *net.UDPAddr) Address {
	return NewIPAndPort(net.IPAddr{IP: a.IP, Zone: a.Zone}, a.Port)
}

func FromTCPAddr(a *net.TCPAddr) Address {
	return NewIPAndPort(net.IPAddr{IP: a.IP, Zone: a.Zone}, a.Port)
}

func AsAddress(addr net.Addr) Address {
	switch a := addr.(type) {
	case *net.TCPAddr:
		return FromTCPAddr(a)
	case *net.UDPAddr:
		return FromUDPAddr(a)
	case *net.IPAddr:
		return NewIP(*a)
	case Address:
		return a
	case nil:
		return Address{}
	default:
		panic(throw.IllegalValue())
	}
}
