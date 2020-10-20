// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nwapi

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nwapi/nwaddr"
)

// NewIP returns an IP address
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

// NewIPAndPort returns a host and port address. Panics on invalid (port).
func NewIPAndPort(ip net.IPAddr, port int) Address {
	if port <= 0 || port > math.MaxUint16 {
		panic(throw.IllegalValue())
	}
	a := NewIP(ip)
	a.port = uint16(port)
	return a
}

// NewHost checks if the string can be parsed into IP address, then returns either IP or DNS values accordingly.
func NewHost(host string) Address {
	if host == "" {
		panic(throw.IllegalValue())
	}
	if ip, zone := parseIPZone(host); ip != nil {
		return newIP(ip, zone)
	}
	return Address{network: uint8(nwaddr.DNS), data1: longbits.WrapStr(strings.ToLower(host))}
}

// parseIPZone mimics net.parseIPZone behavior
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

// NewHostAndPort checks if the string can be parsed into IP address, then returns either IP or DNS values accordingly. Port must be valid.
func NewHostAndPort(host string, port int) Address {
	if port <= 0 || port > math.MaxUint16 {
		panic(throw.IllegalValue())
	}
	a := NewHost(host)
	a.port = uint16(port)
	return a
}

// NewHostPort checks if the string can be parsed into IP address, then returns either IP or DNS values accordingly. Port must be present.
func NewHostPort(hostport string, allowZero bool) Address {
	if hostport == "" {
		panic(throw.IllegalValue())
	}
	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		panic(err)
	}

	switch portN, err := strconv.ParseUint(port, 10, 16); {
	case err != nil:
		panic(err)
	case portN == 0 && !allowZero:
		panic(throw.IllegalValue())
	default:
		a := NewHost(host)
		a.port = uint16(portN)
		return a
	}
}

// NewHostID returns a nodeID based address
func NewHostID(id HostID) Address {
	a := Address{network: uint8(nwaddr.HostID)}
	binary.LittleEndian.PutUint64(a.data0[:], uint64(id))
	return a
}

// NewHostID returns a PK based address
func NewHostPK(pk longbits.FixedReader) Address {
	return Address{network: uint8(nwaddr.HostPK), data1: pk.AsByteString()}
}

// NewLocalUID returns uid-based address. Value of (id) is for convenience only.
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

func (a Address) IdentityType() nwaddr.Identity {
	return nwaddr.Identity(a.network & networkMask)
}

// HostIdentity strips out all extra data, like port or a name of a resolved address
func (a Address) HostIdentity() Address {
	switch {
	case a.port != 0:
		//
	case a.flags&data1isName != 0:
		//
	case a.IdentityType() != nwaddr.LocalUID:
		return a
	// when Addr() == nwaddr.LocalUID, the it may contain HostID
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

// EqualHostIdentity is a convenience method to compare HostIdentity of both addresses
func (a Address) EqualHostIdentity(o Address) bool {
	return a.HostIdentity() == o.HostIdentity()
}

// WithoutName returns address without a symbolic name. This does not change DNS (unresolved) address.
func (a Address) WithoutName() Address {
	if a.flags&data1isName == 0 {
		// optimization
		return a
	}
	a.flags &^= data1isName
	a.data1 = ""
	return a
}

// WithoutPort returns address without a port.
func (a Address) WithoutPort() Address {
	if a.port == 0 {
		// optimization
		return a
	}
	a.port = 0
	return a
}

// WithPort returns address with the given (port).
func (a Address) WithPort(port uint16) Address {
	if port == 0 {
		panic(throw.IllegalValue())
	}

	if a.port == port {
		// optimization
		return a
	}
	a.port = port
	return a
}

// Network enables use of Address as net.Addr, but will only be valid for IP/DNS identity
func (a Address) Network() string {
	switch a.IdentityType() {
	case nwaddr.DNS, nwaddr.IP:
		return "ip"
	default:
		return "invalid"
	}
}

// String provides value compatible with net.Addr, but will only be valid for IP/DNS identity
func (a Address) String() string {
	h := a.HostString()
	if !a.HasPort() {
		return h
	}
	p := strconv.Itoa(int(a.port))
	if a.IdentityType() == nwaddr.IP {
		return net.JoinHostPort(h, p)
	}
	return h + ":" + p
}

// HostString provides a readable string that describes the address/host. This string doesn't include port.
func (a Address) HostString() string {
	switch a.IdentityType() {
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

// Resolve applies the given (resolver) to this address only when this address is unresolved. One address is picked by the (preference).
func (a Address) Resolve(ctx context.Context, resolver BasicAddressResolver, preference Preference) (Address, error) {
	ips, err := a.ResolveAll(ctx, resolver)
	if err != nil {
		return Address{}, err
	}
	return preference.ChooseOne(ips), nil
}

// ResolveAll applies the given (resolver) to this address only when this address is unresolved.
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
		aa[i] = NewIP(ips[i])
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

	switch n := a.IdentityType(); {
	case n == nwaddr.DNS:
		return resolver.LookupIPAddr(ctx, string(a.data1))
	default:
		if r, ok := resolver.(AddressResolver); ok {
			return r.LookupNetworkAddress(ctx, a)
		}
	}
	return nil, throw.NotImplemented()
}

// HasPort returns true when a valid port is present
func (a Address) HasPort() bool {
	return a.port > 0
}

// Port returns a valid port or panics
func (a Address) Port() int {
	if a.HasPort() {
		return int(a.port)
	}
	panic(throw.IllegalState())
}

// IsResolved indicates that this address doesn't require a resolver
func (a Address) IsResolved() bool {
	return a.IdentityType().IsResolved()
}

// IsResolved indicates that this address can be used for net.Addr
func (a Address) IsNetCompatible() bool {
	return a.IdentityType().IsNetCompatible()
}

// IsResolved indicates that this address can be connected
func (a Address) CanConnect() bool {
	return a.IdentityType().IsResolved() && a.HasPort()
}

// AsUDPAddr returns net.UDPAddr or panics
func (a *Address) AsUDPAddr() net.UDPAddr {
	// ptr receiver is to prevent copy on slice op
	return net.UDPAddr{
		IP:   a.asIP(),
		Port: int(a.port),
		Zone: a.getIPv6Zone(),
	}
}

// AsTCPAddr returns net.TCPAddr or panics
func (a *Address) AsTCPAddr() net.TCPAddr {
	// ptr receiver is to prevent copy on slice op
	return net.TCPAddr{
		IP:   a.asIP(),
		Port: int(a.port),
		Zone: a.getIPv6Zone(),
	}
}

// AsIPAddr returns net.IPAddr or panics
func (a *Address) AsIPAddr() net.IPAddr {
	// ptr receiver is to prevent copy on slice op
	return net.IPAddr{
		IP:   a.asIP(),
		Zone: a.getIPv6Zone(),
	}
}

func (a *Address) asIP() net.IP {
	// ptr receiver is to prevent copy on slice op
	if a.IdentityType() == nwaddr.IP {
		return a.data0[:net.IPv6len]
	}
	panic(throw.IllegalState())
}

// Data provides a binary form of this address, only host part, excludes port
func (a Address) Data() longbits.ByteString {
	switch a.IdentityType() {
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

// Data provides size of the a binary form
func (a Address) DataLen() int {
	switch a.IdentityType() {
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
	if a.IdentityType() == nwaddr.IP && a.flags&data1isName == 0 && !a.isIPv4Prefix() {
		return string(a.data1)
	}
	return ""
}

// HasName returns true when this address has a textual name
func (a Address) HasName() bool {
	return a.IdentityType() == nwaddr.DNS || a.flags&data1isName != 0
}

// Name returns a textual name or panics
func (a Address) Name() string {
	if a.HasName() {
		return string(a.data1)
	}
	panic(throw.IllegalState())
}

// AsHostID returns HostID or panics
func (a Address) AsHostID() HostID {
	switch a.IdentityType() {
	case nwaddr.HostID, nwaddr.LocalUID:
		return HostID(binary.LittleEndian.Uint64(a.data0[:8]))
	}
	panic(throw.IllegalState())
}

// AsLocalUID returns LocalUniqueID or panics
func (a Address) AsLocalUID() LocalUniqueID {
	if a.IdentityType() == nwaddr.LocalUID {
		return LocalUniqueID(binary.LittleEndian.Uint64(a.data0[8:]))
	}
	panic(throw.IllegalState())
}

// IsLoopback returns true when address is IP and it is a loopback
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

func (a *Address) ProtoSize() int {
	return 2 + 1 + 1 + 16 + len(a.data1)
}

func (a *Address) MarshalTo(data []byte) (int, error) {
	buffer := bytes.NewBuffer(make([]byte, 0))

	if err := binary.Write(buffer, binary.BigEndian, a.port); err != nil {
		return 0, err
	}

	if err := binary.Write(buffer, binary.BigEndian, a.network); err != nil {
		return 0, err
	}

	if err := binary.Write(buffer, binary.BigEndian, a.flags); err != nil {
		return 0, throw.W(err, "failed to marshal protobuf host flags")
	}

	_, err := a.data0.WriteTo(buffer)
	if err != nil {
		return 0, err
	}

	_, err = a.data1.WriteTo(buffer)
	if err != nil {
		return 0, err
	}

	copy(data, buffer.Bytes())
	fmt.Println(data)

	return buffer.Len(), nil
}

func (a *Address) Unmarshal(data []byte) error {
	fmt.Println(data)
	reader := bytes.NewReader(data)
	if err := binary.Read(reader, binary.BigEndian, &a.port); err != nil {
		return err
	}

	if err := binary.Read(reader, binary.BigEndian, &a.network); err != nil {
		return err
	}

	if err := binary.Read(reader, binary.BigEndian, &a.flags); err != nil {
		return err
	}

	// if err := binary.Read(reader, binary.BigEndian, &a.data0); err != nil {
	// 	return err
	// }

	// a.data1 = longbits.CopyBytes(data[20:])

	// panic(fmt.Sprintf("unmarshall address: %s", a.String()))
	return nil
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
