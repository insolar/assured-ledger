// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l2

import "github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/apinetwork"

type ConnectionMode uint32

const (
	AllowUnknownPeer ConnectionMode = iota
	//AllowIncomingConnections
)

func (v ConnectionMode) IsProtocolAllowed(prot apinetwork.ProtocolType) bool {
	return v&1<<(prot+16) != 0
}

func (v ConnectionMode) IsUnknownPeerAllowed() bool {
	return v&AllowUnknownPeer != 0
}

//func (v ConnectionMode) IsIncomingAllowed() bool {
//	return v & AllowIncomingConnections != 0
//}
