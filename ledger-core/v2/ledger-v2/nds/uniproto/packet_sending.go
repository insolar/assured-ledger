// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniproto

import "github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"

type SenderPacket struct {
	Packet
	signer    PacketDataSigner
	encrypter cryptkit.Encrypter
}

type PacketDataSigner struct {
	signer cryptkit.DataSigner
}
