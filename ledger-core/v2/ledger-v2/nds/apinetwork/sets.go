// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package apinetwork

func NewProtocolSet(pts ...ProtocolType) ProtocolSet {
	v := ProtocolSet(0)
	for _, pt := range pts {
		v = v.Set(pt, true)
	}
	return v
}

type ProtocolSet uint16

const AllProtocols ProtocolSet = ^ProtocolSet(0)

func (v ProtocolSet) Has(pt ProtocolType) bool {
	return v&1<<pt != 0
}

func (v ProtocolSet) Set(pt ProtocolType, val bool) ProtocolSet {
	if val {
		return v | 1<<pt
	}
	return v &^ 1 << pt
}

func NewPacketSet(pts ...ProtocolType) ProtocolSet {
	v := ProtocolSet(0)
	for _, pt := range pts {
		v = v.Set(pt, true)
	}
	return v
}

type PacketSet uint16

const AllPackets PacketSet = ^PacketSet(0)

func (v PacketSet) Has(pt uint8) bool {
	return v&1<<pt != 0
}

func (v PacketSet) Set(pt uint8, val bool) PacketSet {
	if val {
		return v | 1<<pt
	}
	return v &^ 1 << pt
}
