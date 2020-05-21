// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package host

import (
	"net"

	errors "github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

// Address is host's real network address.
type Address struct {
	net.UDPAddr
}

// NewAddress is constructor.
func NewAddress(address string) (*Address, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, errors.W(err, "Failed to resolve ip address")
	}
	return &Address{UDPAddr: *udpAddr}, nil
}

// Equal checks if address is equal to another.
func (address Address) Equal(other Address) bool {
	return address.IP.Equal(other.IP) && address.Port == other.Port
}
