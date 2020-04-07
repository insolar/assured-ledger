// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l1

import (
	"context"
	"encoding/binary"
	"math"
	"net"
	"strconv"
	"strings"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/apinetwork"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

func NewIP(ip net.IPAddr) Address {
	return newIP(ip.IP, ip.Zone)
}

func newIP(ip net.IP, zone string) Address {
	a := Address{network: byte(IP)}
	switch len(ip) {
	case net.IPv6len:
		copy(a.data0[:], ip)
		switch {
		case zone == "":
			//
		case a.isIPv4Prefix():
			panic(throw.IllegalValue())
		default:
			a.data1 = longbits.WrapStr(zone)
		}
	case net.IPv4len:
		copy(a.data0[:], v4InV6Prefix)
		copy(a.data0[len(v4InV6Prefix):], ip)
	default:
		panic(throw.IllegalValue())
	}
	return a
}

func NewIPAndPort(ip net.IPAddr, port int) Address {
	if port <= 0 || port > math.MaxUint16 {
		panic(throw.IllegalValue())
	}
	a := NewIP(ip)
	a.port = uint16(port)
	return a
}

func NewHost(host string) Address {
	if host == "" {
		panic(throw.IllegalValue())
	}
	if ip, zone := parseIPZone(host); ip != nil {
		return newIP(ip, zone)
	}
	return Address{network: uint8(DNS), data1: longbits.WrapStr(strings.ToLower(host))}
}

// Mimics net.parseIPZone behavior
func parseIPZone(s string) (net.IP, string) {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '.':
			return net.ParseIP(s), ""
		case ':':
			z := ""
			if i := strings.LastIndexByte(s, '%'); i > 0 {
				z = s[i+1:]
				s = s[:i]
			}
			return net.ParseIP(s), z
		}
	}
	return nil, ""
}

func NewHostAndPort(host string, port int) Address {
	if port <= 0 || port > math.MaxUint16 {
		panic(throw.IllegalValue())
	}
	a := NewHost(host)
	a.port = uint16(port)
	return a
}

func NewHostPort(hostport string) Address {
	if hostport == "" {
		panic(throw.IllegalValue())
	}
	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		panic(err)
	}

	if portN, err := strconv.ParseUint(port, 10, 16); err != nil {
		panic(err)
	} else if portN == 0 {
		panic(throw.IllegalValue())
	} else {
		a := NewHost(host)
		a.port = uint16(portN)
		return a
	}
}

var _ net.Addr = Address{}

type Address struct {
	data0   longbits.Bits128
	data1   longbits.ByteString
	port    uint16
	network uint8
	flags   addressFlags
}

type addressFlags uint8

const (
	networkBits = 4
	networkMask = 1<<networkBits - 1
)

const (
	data1isDNS addressFlags = 1 << iota
)

func (a Address) IsZero() bool {
	return a.network == 0
}

func (a Address) AddrNetwork() AddressNetwork {
	return AddressNetwork(a.network & networkMask)
}

func (a Address) Network() string {
	switch a.AddrNetwork() {
	case DNS, IP:
		return "ip"
	default:
		return "invalid"
	}
}

func (a Address) data0Len() uint8 {
	return a.network>>networkBits + 1
}

func (a Address) HostOnly() Address {
	if a.port == 0 {
		// optimization
		return a
	}
	a.port = 0
	return a
}

func (a Address) WithPort(port uint16) Address {
	if port <= 0 || port > math.MaxUint16 {
		panic(throw.IllegalValue())
	}

	if a.port == port {
		// optimization
		return a
	}
	a.port = port
	return a
}

type AddressIdentity string

//func (a Address) ExactIdentity() AddressIdentity {
//	prefix := ""
//	switch a.Network() {
//	case 0:
//		return ""
//	case DNS:
//		prefix = "N"
//	case IP:
//		prefix = "I" // take data0!
//	case HostPK:
//		prefix = "P"
//	case HostID:
//		prefix = "#"
//	default:
//		prefix = strconv.Itoa(int(a.network)) + "?"
//	}
//	return AddressIdentity(prefix + string(a.data1))
//}
//
//func (a Address) CoarseIdentity() uint64 {
//	switch a.Network() {
//	case 0:
//		return 0
//	case DNS:
//		return uint64(aeshash.ByteStr(a.data1))
//	case IP:
//		return a.data0.FoldToUint64()
//	default:
//		if len(a.data1) > 0 {
//			return uint64(aeshash.ByteStr(a.data1))
//		}
//		return a.data0.FoldToUint64()
//	}
//}

func (a Address) String() string {
	h := a.HostString()
	if !a.HasPort() {
		return h
	}
	p := strconv.Itoa(int(a.port))
	if a.AddrNetwork() == IP {
		return net.JoinHostPort(h, p)
	}
	return h + ":" + p
}

func (a Address) HostString() string {
	switch a.AddrNetwork() {
	case 0:
		return ""
	case DNS:
		return string(a.data1)
	case IP:
		return a.AsIPAddr().String()
	case HostPK:
		return "(PK)" + a.data1.Hex()
	case HostID:
		return "#" + a.AsHostId().String()
	default:
		return "(" + strconv.Itoa(int(a.network)) + ")" + a.data1.String()
	}
}

func (a Address) Resolve(ctx context.Context, resolver BasicAddressResolver) (Address, error) {
	if a.IsResolved() {
		return a, nil
	}
	ips, err := a.resolve(ctx, resolver)
	if err != nil {
		return Address{}, err
	}

	aa := NewIP(ips[0])
	aa.port = a.port
	return aa, nil
}

func (a Address) ResolveAll(ctx context.Context, resolver BasicAddressResolver) ([]Address, error) {
	if a.IsResolved() {
		return []Address{a}, nil
	}
	ips, err := a.resolve(ctx, resolver)
	if err != nil {
		return nil, err
	}

	aa := make([]Address, len(ips))
	for i := range ips {
		aa[i] = NewIP(ips[0])
		aa[i].port = a.port
	}
	return aa, nil
}

func (a Address) resolve(ctx context.Context, resolver BasicAddressResolver) ([]net.IPAddr, error) {
	switch {
	case a.IsZero():
		panic(throw.IllegalState())
	case resolver == nil:
		panic(throw.IllegalValue())
	}

	switch n := a.AddrNetwork(); {
	//case n.IsResolved():
	//	return []net.IPAddr{a.AsIPAddr()}, nil
	case n == DNS:
		return resolver.LookupIPAddr(ctx, string(a.data1))
	default:
		if r, ok := resolver.(AddressResolver); ok {
			return r.LookupNetworkAddress(ctx, a)
		}
	}
	return nil, throw.NotImplemented()
}

func (a Address) HasPort() bool {
	return a.port > 0
}

func (a Address) Port() int {
	if a.HasPort() {
		return int(a.port)
	}
	panic(throw.IllegalState())
}

func (a Address) IsResolved() bool {
	return a.AddrNetwork().IsResolved()
}

func (a Address) IsNetCompatible() bool {
	return a.AddrNetwork().IsNetCompatible()
}

func (a Address) CanConnect() bool {
	return a.AddrNetwork().IsResolved() && a.HasPort()
}

// ptr receiver is to prevent copy on slice op
func (a *Address) AsUDPAddr() net.UDPAddr {
	return net.UDPAddr{
		IP:   a.asIP(),
		Port: a.Port(),
		Zone: a.getIPv6Zone(),
	}
}

// ptr receiver is to prevent copy on slice op
func (a *Address) AsTCPAddr() net.TCPAddr {
	return net.TCPAddr{
		IP:   a.asIP(),
		Port: a.Port(),
		Zone: a.getIPv6Zone(),
	}
}

// ptr receiver is to prevent copy on slice op
func (a *Address) AsIPAddr() net.IPAddr {
	return net.IPAddr{
		IP:   a.asIP(),
		Zone: a.getIPv6Zone(),
	}
}

// ptr receiver is to prevent copy on slice op
func (a *Address) asIP() net.IP {
	if a.AddrNetwork() == IP {
		return a.data0[:net.IPv6len]
	}
	panic(throw.IllegalState())
}

func (a Address) Data() longbits.ByteString {
	switch a.AddrNetwork() {
	case 0:
		return ""
	case DNS:
		return a.data1
	case IP:
		return longbits.CopyBytes(a.data0[:net.IPv6len])
	default:
		if len(a.data1) > 0 {
			return a.data1
		}
		return longbits.CopyBytes(a.data0[:a.data0Len()])
	}
}

func (a Address) DataLen() int {
	switch a.AddrNetwork() {
	case 0:
		return 0
	case DNS:
		return len(a.data1)
	case IP:
		return net.IPv6len
	case HostID:
		return apinetwork.HostIdByteSize
	default:
		if len(a.data1) > 0 {
			return len(a.data1)
		}
		return int(a.data0Len())
	}
}

func (a Address) AsUint8() uint8 {
	if len(a.data1) == 0 && a.data0Len() == 1 {
		return a.data0[0]
	}
	panic(throw.IllegalState())
}

func (a Address) AsUint16() uint16 {
	if len(a.data1) == 0 && a.data0Len() == 2 {
		return binary.LittleEndian.Uint16(a.data0[:])
	}
	panic(throw.IllegalState())
}

func (a Address) AsUint32() uint32 {
	if len(a.data1) == 0 && a.data0Len() == 4 {
		return binary.LittleEndian.Uint32(a.data0[:])
	}
	panic(throw.IllegalState())
}

func (a Address) AsUint64() uint64 {
	if len(a.data1) == 0 && a.data0Len() == 8 {
		return binary.LittleEndian.Uint64(a.data0[:])
	}
	panic(throw.IllegalState())
}

func (a Address) AsBits128() longbits.Bits128 {
	if len(a.data1) == 0 && a.data0Len() == 16 {
		return a.data0
	}
	panic(throw.IllegalState())
}

const v4InV6Prefix = "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xFF\xFF"

func (a Address) isIPv4() bool {
	return a.network == uint8(IP) && a.isIPv4Prefix()
}

func (a Address) isIPv6() bool {
	return a.network == uint8(IP) && !a.isIPv4Prefix()
}

func (a Address) isIPv4Prefix() bool {
	return v4InV6Prefix == string(a.data0[:len(v4InV6Prefix)])
}

func (a Address) getIPv6Zone() string {
	if a.AddrNetwork() == IP && a.flags&data1isDNS == 0 && !a.isIPv4Prefix() {
		return string(a.data1)
	}
	return ""
}

func (a Address) HasName() bool {
	return a.AddrNetwork() == DNS || a.flags&data1isDNS != 0
}

func (a Address) Name() string {
	if a.HasName() {
		return string(a.data1)
	}
	panic(throw.IllegalState())
}

func (a Address) AsHostId() apinetwork.HostId {
	if a.AddrNetwork() == HostID {
		return apinetwork.HostId(binary.LittleEndian.Uint64(a.data0[:]))
	}
	panic(throw.IllegalState())
}

/********************************************************/

func Join(addresses ...[]Address) []Address {
	switch len(addresses) {
	case 0:
		return nil
	case 1:
		return addresses[0]
	}

	result := make([]Address, 0, len(addresses[0]))

	// this is cheaper than map[]bool
	m := make(map[Address]struct{}, len(result)<<1)

	for _, set := range addresses {
		for _, a := range set {
			if _, ok := m[a]; !ok {
				m[a] = struct{}{}
				result = append(result, a)
			}
		}
	}
	return result
}
