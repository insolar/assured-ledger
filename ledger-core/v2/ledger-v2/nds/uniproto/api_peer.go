// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniproto

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
)

type Peer interface {
	GetPrimary() nwapi.Address
	GetSignatureKey() cryptkit.SignatureKey
	GetNodeID() nwapi.ShortNodeID

	SetProtoInfo(pt ProtocolType, info io.Closer)
	GetProtoInfo(pt ProtocolType) io.Closer
	GetOrCreateProtoInfo(pt ProtocolType, factoryFn func(Peer) io.Closer) io.Closer
	Transport() OutTransport
}
