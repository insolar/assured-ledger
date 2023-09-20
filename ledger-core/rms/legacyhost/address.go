package legacyhost

import (
	"net"

	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

// Address is host's real network address.
type Address struct {
	net.UDPAddr
}

// NewAddress is constructor.
func NewAddress(address string) (Address, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return Address{}, errors.W(err, "Failed to resolve ip address")
	}
	return Address{UDPAddr: *udpAddr}, nil
}

func (address Address) IsZero() bool {
	return len(address.IP) == 0
}

// Equal checks if address is equal to another.
func (address Address) Equal(other Address) bool {
	return address.IP.Equal(other.IP) && address.Port == other.Port
}
