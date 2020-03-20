// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l2

import "github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"

func NewMessageUdp(local Address, receiver TransportReceiveFunc) Transport {
	panic(throw.NotImplemented())
}

func NewMessageTcp(local Address, receiver TransportReceiveFunc, tls TlsConfig) Transport {
	panic(throw.NotImplemented())
}

func NewLargeTcp(local Address, receiver TransportReceiveFunc, tls TlsConfig) Transport {
	panic(throw.NotImplemented())
}

type TlsConfig struct {
}
