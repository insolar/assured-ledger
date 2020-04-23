// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nwaddr

type Network uint8

const (
	_ Network = iota
	IP
	DNS
	HostPK
	HostID
	LocalUID
)

func (a Network) IsIP() bool {
	return a == IP
}

func (a Network) IsNetCompatible() bool {
	return a == IP || a == DNS
}

func (a Network) IsResolved() bool {
	return a == IP
}
