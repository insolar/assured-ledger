// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"strconv"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

func NewDirectAddress(id nwapi.ShortNodeID) DeliveryAddress {
	if id.IsAbsent() {
		panic(throw.IllegalValue())
	}
	return DeliveryAddress{
		addrType:     DirectAddress,
		nodeSelector: uint32(id),
	}
}

func NewRoleAddress(roleID uint8, dataSelector uint64) DeliveryAddress {
	if roleID == 0 {
		panic(throw.IllegalValue())
	}
	return DeliveryAddress{
		addrType:     RoleAddress,
		nodeSelector: uint32(roleID),
		dataSelector: dataSelector,
	}
}

type ResolverFunc func(addrType AddressFlags, nodeSelector uint32, dataSelector uint64) nwapi.Address

type DeliveryAddress struct {
	addrType     AddressFlags
	nodeSelector uint32
	dataSelector uint64
}

func (v DeliveryAddress) IsZero() bool {
	return v.addrType == 0 && v.nodeSelector == 0
}

func (v DeliveryAddress) AsDirect() nwapi.Address {
	if v.addrType != DirectAddress {
		panic(throw.IllegalState())
	}
	return v.ResolveWith(nil)
}

func (v DeliveryAddress) ResolveWith(fn ResolverFunc) nwapi.Address {
	if v.addrType == DirectAddress {
		switch {
		case v.nodeSelector == 0:
			panic(throw.IllegalValue())
		case v.dataSelector != 0:
			panic(throw.IllegalValue())
		}
		return nwapi.NewHostID(nwapi.HostID(v.nodeSelector))
	}

	return fn(v.addrType, v.nodeSelector, v.dataSelector)
}

// func (v DeliveryAddress) String() string {
//	// TODO DeliveryAddress.String()
// }

type AddressFlags uint32

const (
	DirectAddress AddressFlags = 0
	RoleAddress   AddressFlags = 1 << iota
)

type ReturnAddress struct {
	returnTo nwapi.Address
	returnID ShortShipmentID
	expires  uint32
	canPull  bool
}

func (v ReturnAddress) IsZero() bool {
	return v.returnTo.IsZero()
}

func (v ReturnAddress) IsValid() bool {
	return !v.returnTo.IsZero() && v.returnID != 0
}

func (v ReturnAddress) String() string {
	return v.returnTo.String() + "/" + strconv.FormatUint(uint64(v.returnID), 10)
}
