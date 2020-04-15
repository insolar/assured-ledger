// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l2

import (
	"math"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/uniproto"
)

type ConnectionMode uint32

const (
	AllowUnknownPeer ConnectionMode = 1 << iota
)

const AllowAll = ^ConnectionMode(0)

func (v ConnectionMode) IsProtocolAllowed(pt uniproto.ProtocolType) bool {
	return v&1<<(pt+16) != 0
}

func (v ConnectionMode) IsUnknownPeerAllowed() bool {
	return v&AllowUnknownPeer != 0
}

func (v ConnectionMode) AllowedSet() uniproto.ProtocolSet {
	return uniproto.ProtocolSet(v >> 16)
}

func (v ConnectionMode) SetAllowedSet(s uniproto.ProtocolSet) ConnectionMode {
	return v&math.MaxUint16 | ConnectionMode(s)<<16
}
