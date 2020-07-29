// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package endpoints

import (
	"bytes"
	"encoding/binary"
	"net"
	"strconv"

	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

const (
	ipSize        = net.IPv6len
	portSize      = 2
	ipAddressSize = ipSize + portSize

	maxPortNumber = ^uint16(0)
)

var defaultByteOrder = binary.BigEndian

type IPAddress [ipAddressSize]byte

func NewIPAddress(address string) (IPAddress, error) {
	return _newIPAddress(address, false)
}

func NewIPAddressZeroPort(address string) (IPAddress, error) {
	return _newIPAddress(address, true)
}

func _newIPAddress(address string, allowZero bool) (IPAddress, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return IPAddress{}, errors.Errorf("invalid address: %s", address)
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return IPAddress{}, errors.Errorf("invalid ip: %s", host)
	}

	portNumber, err := strconv.Atoi(port)
	switch {
	case err != nil:
		return IPAddress{}, errors.Errorf("invalid port number: %s", port)
	case portNumber > 0 && portNumber <= int(maxPortNumber):
	case allowZero && portNumber == 0:
	default:
		return IPAddress{}, errors.Errorf("invalid port number: %d", portNumber)
	}

	return newIPAddress(ip, uint16(portNumber))
}


func newIPAddress(ip net.IP, portNumber uint16) (addr IPAddress, err error) {
	switch ipSize {
	case net.IPv6len:
		ip = ip.To16()
	case net.IPv4len:
		ip = ip.To4()
	default:
		panic("not implemented")
	}

	portBytes := make([]byte, portSize)
	defaultByteOrder.PutUint16(portBytes, portNumber)

	copy(addr[:], ip)
	copy(addr[ipSize:], portBytes)

	return
}

func (a IPAddress) String() string {
	r := bytes.NewReader(a[:])

	ipBytes := make([]byte, ipSize)
	_, _ = r.Read(ipBytes)

	portBytes := make([]byte, portSize)
	_, _ = r.Read(portBytes)

	ip := net.IP(ipBytes)
	portNumber := defaultByteOrder.Uint16(portBytes)

	host := ip.String()
	port := strconv.Itoa(int(portNumber))

	return net.JoinHostPort(host, port)
}
