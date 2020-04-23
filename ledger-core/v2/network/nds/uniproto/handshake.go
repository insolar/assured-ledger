// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniproto

import (
	"encoding/binary"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

/*
	ByteSize=16
*/
type HandshakeHeader struct {
	// HandshakeHeader is compatible with Header, but uses length = 0, which is invalid for a normal header

	// Magic   [3]uint8 = "INS"
	Phase uint8
	// _    uint8          `insolar-transport:"[:]=0"`
	Flags   HandshakeFlags `insolar-transport:"[0]=1;[1]=0;[2]=IsLarge"`
	Version uint8          `insolar-transport:"[0:15]=1..15"` // value MUST be less than HeaderByteSizeMin
	// _    uint8
	Random uint64
}

const HandshakeMagic = "INS"

type HandshakeFlags uint8

const (
	_handshakeMustBeSet0 HandshakeFlags = 1 << iota
	_handshakeMustBeUnset0
	HandshakeLarge

	handshakeFlagsMask = (1<<iota - 1) ^ _handshakeMustBeUnset0
	handshakeOneMask   = _handshakeMustBeSet0
)

const (
	HandshakeMinVersion = 1
	HandshakeMaxVersion = HeaderByteSizeMin - 1

	HandshakeMaxPhase = 0
)

func (p *HandshakeHeader) Deserialize(b []byte) int {
	switch {
	case len(b) < HeaderByteSizeMin:
		//
	case string(b[:2]) != HandshakeMagic:
		//
	case b[4] != 0 || b[7] != 0:
		//
	default:
		p.Phase = b[3]
		p.Flags = HandshakeFlags(b[5])
		p.Version = b[6]
		switch {
		case p.Version < HandshakeMinVersion || p.Version > HandshakeMaxVersion:
			//
		case p.Flags&handshakeOneMask != handshakeOneMask:
			//
		case p.Flags & ^handshakeFlagsMask != 0:
			//
		default:
			p.Random = binary.LittleEndian.Uint64(b[8:])
			return HeaderByteSizeMin
		}
	}
	return 0
}

func (p *HandshakeHeader) Serialize(b []byte) int {
	switch {
	case p.Version < HandshakeMinVersion:
		panic(throw.IllegalState())
	case p.Version > HandshakeMaxVersion:
		panic(throw.IllegalState())
	}

	binary.LittleEndian.PutUint64(b[8:], p.Random) // + range check
	copy(b[:2], HandshakeMagic)
	b[3] = p.Phase
	b[4] = 0
	b[5] = byte(handshakeOneMask | p.Flags&handshakeFlagsMask)
	b[6] = p.Version
	b[7] = 0
	return HeaderByteSizeMin
}
