// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nwapi

type AddressNetwork uint8

const (
	_ AddressNetwork = iota
	IP
	DNS
	HostID
	HostPK
	LocalUID
)

func (a AddressNetwork) IsIP() bool {
	return a == IP
}

func (a AddressNetwork) IsNetCompatible() bool {
	return a == IP || a == DNS
}

func (a AddressNetwork) IsResolved() bool {
	return a == IP
}
