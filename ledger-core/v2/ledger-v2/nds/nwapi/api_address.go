// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nwapi

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"math"
	"net"
	"strconv"
	"strings"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/nwapi/nwaddr"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

func NewIP(ip net.IPAddr) Address {
	return newIP(ip.IP, ip.Zone)
}

func newIP(ip net.IP, zone string) Address {
	a := Address{network: byte(nwaddr.IP)}
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
	return Address{network: uint8(nwaddr.DNS), data1: longbits.WrapStr(strings.ToLower(host))}
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

func NewHostID(id HostID) Address {
	a := Address{network: uint8(nwaddr.HostID)}
	binary.LittleEndian.PutUint64(a.data0[:], uint64(id))
	return a
}

func NewHostPK(pk longbits.FixedReader) Address {
	return Address{network: uint8(nwaddr.HostPK), data1: pk.AsByteString()}
}

func NewLocalUID(uid uint64, id HostID) Address {
	if uid == 0 {
		panic(throw.IllegalState())
	}
	a := Address{network: uint8(nwaddr.LocalUID)}
	binary.LittleEndian.PutUint64(a.data0[:8], uint64(id))
	binary.LittleEndian.PutUint64(a.data0[8:], uid)
	return a
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
	data1isName addressFlags = 1 << iota
)

func (a Address) IsZero() bool {
	return a.network == 0
}

func (a Address) AddrNetwork() nwaddr.Network {
	return nwaddr.Network(a.network & networkMask)
}

func (a Address) Network() string {
	switch a.AddrNetwork() {
	case nwaddr.DNS, nwaddr.IP:
		return "ip"
	default:
		return "invalid"
	}
}

func (a Address) HostIdentity() Address {
	switch {
	case a.port != 0:
		//
	case a.flags&data1isName != 0:
		//
	case a.AddrNetwork() != nwaddr.LocalUID:
		return a
	case string(a.data0[:8]) == zero8Prefix:
		return a
	default:
		copy(a.data0[:8], zero8Prefix)
	}

	a.port = 0
	a.flags &^= data1isName
	a.data1 = ""
	return a
}

func (a Address) WithoutName() Address {
	if a.flags&data1isName == 0 {
		// optimization
		return a
	}
	a.flags &^= data1isName
	a.data1 = ""
	return a
}

func (a Address) WithoutPort() Address {
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

func (a Address) String() string {
	h := a.HostString()
	if !a.HasPort() {
		return h
	}
	p := strconv.Itoa(int(a.port))
	if a.AddrNetwork() == nwaddr.IP {
		return net.JoinHostPort(h, p)
	}
	return h + ":" + p
}

func (a Address) HostString() string {
	switch a.AddrNetwork() {
	case 0:
		return ""
	case nwaddr.DNS:
		return string(a.data1)
	case nwaddr.IP:
		ipa := a.AsIPAddr()
		return ipa.String()
	case nwaddr.HostPK:
		return "(PK)" + a.data1.Hex()
	case nwaddr.HostID:
		return "#" + a.AsHostID().String()
	case nwaddr.LocalUID:
		hid := a.AsHostID()
		s := ""
		if !hid.IsAbsent() {
			s = "[#" + hid.String() + "]"
		}
		return "(UID)" + hex.EncodeToString(a.data0[8:]) + s
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

	// TODO add host name to data1

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

	// TODO add host name to data1

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
	case n == nwaddr.DNS:
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
	if a.AddrNetwork() == nwaddr.IP {
		return a.data0[:net.IPv6len]
	}
	panic(throw.IllegalState())
}

func (a Address) Data() longbits.ByteString {
	switch a.AddrNetwork() {
	case 0:
		return ""
	case nwaddr.DNS:
		return a.data1
	case nwaddr.IP:
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
	case nwaddr.DNS:
		return len(a.data1)
	case nwaddr.IP:
		return net.IPv6len
	case nwaddr.HostID:
		return HostIDByteSize
	case nwaddr.LocalUID:
		return len(a.data0)
	default:
		if len(a.data1) > 0 {
			return len(a.data1)
		}
		return int(a.data0Len())
	}
}

func (a Address) data0Len() uint8 {
	n := a.network >> networkBits
	if n == 0 {
		return 0
	}
	return n + 1
}

//func (a Address) AsUint16() uint16 {
//	if a.data0Len() == 2 {
//		return binary.LittleEndian.Uint16(a.data0[:])
//	}
//	panic(throw.IllegalState())
//}
//
//func (a Address) AsUint32() uint32 {
//	if a.data0Len() == 4 {
//		return binary.LittleEndian.Uint32(a.data0[:])
//	}
//	panic(throw.IllegalState())
//}
//
//func (a Address) AsUint64() uint64 {
//	if a.data0Len() == 8 {
//		return binary.LittleEndian.Uint64(a.data0[:])
//	}
//	panic(throw.IllegalState())
//}
//
//func (a Address) AsBits128() longbits.Bits128 {
//	if a.data0Len() == 16 {
//		return a.data0
//	}
//	panic(throw.IllegalState())
//}

const zero4Prefix = "\x00\x00\x00\x00"
const zero8Prefix = zero4Prefix + zero4Prefix
const v4InV6Prefix = zero8Prefix + "\x00\x00\xFF\xFF"
const v6Loopback = zero8Prefix + zero4Prefix + "\x00\x00\x00\x01"

func (a Address) isIPv4() bool {
	return a.network == uint8(nwaddr.IP) && a.isIPv4Prefix()
}

func (a Address) isIPv6() bool {
	return a.network == uint8(nwaddr.IP) && !a.isIPv4Prefix()
}

func (a Address) isIPv4Prefix() bool {
	return v4InV6Prefix == string(a.data0[:len(v4InV6Prefix)])
}

func (a Address) getIPv6Zone() string {
	if a.AddrNetwork() == nwaddr.IP && a.flags&data1isName == 0 && !a.isIPv4Prefix() {
		return string(a.data1)
	}
	return ""
}

func (a Address) HasName() bool {
	return a.AddrNetwork() == nwaddr.DNS || a.flags&data1isName != 0
}

func (a Address) Name() string {
	if a.HasName() {
		return string(a.data1)
	}
	panic(throw.IllegalState())
}

func (a Address) AsHostID() HostID {
	switch a.AddrNetwork() {
	case nwaddr.HostID, nwaddr.LocalUID:
		return HostID(binary.LittleEndian.Uint64(a.data0[:8]))
	}
	panic(throw.IllegalState())
}

func (a Address) AsLocalUID() LocalUniqueID {
	if a.AddrNetwork() == nwaddr.LocalUID {
		return LocalUniqueID(binary.LittleEndian.Uint64(a.data0[8:]))
	}
	panic(throw.IllegalState())
}

func (a Address) IsLoopback() bool {
	switch {
	case a.network != uint8(nwaddr.IP):
		return false
	case a.isIPv4Prefix():
		return a.data0[12] == 127
	default:
		return string(a.data0[:]) == v6Loopback
	}
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
