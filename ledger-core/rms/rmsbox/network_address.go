// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rmsbox

import (
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

type NetworkAddress struct {
	address nwapi.Address
}

func NewNetworkAddress(address nwapi.Address) NetworkAddress {
	return NetworkAddress{address: address}
}

func (na *NetworkAddress) Get() nwapi.Address {
	return na.address
}

func (na *NetworkAddress) Set(address nwapi.Address) {
	na.address = address
}

func (na *NetworkAddress) MarshalToSizedBuffer(data []byte) (int, error) {
	if len(data) == 0 {
		panic("empty data")
	}
	return na.address.MarshalTo(data)
}

func (na *NetworkAddress) Unmarshal(data []byte) error {
	return na.address.Unmarshal(data)
}

func (na *NetworkAddress) ProtoSize() int {
	return na.address.ProtoSize()
}
