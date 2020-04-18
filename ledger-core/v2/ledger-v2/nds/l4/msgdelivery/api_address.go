// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"strconv"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

func NewDirectAddress(id nwapi.ShortNodeID) DeliveryAddress {
	if id.IsAbsent() {
		panic(throw.IllegalValue())
	}
	return DeliveryAddress{addrType: DirectAddress, nodeSelector: uint32(id)}
}

func NewRoleAddress(roleId uint8, dataSelector uint64) DeliveryAddress {
	if roleId == 0 {
		panic(throw.IllegalValue())
	}
	return DeliveryAddress{addrType: RoleAddress, nodeSelector: uint32(roleId), dataSelector: dataSelector}
}

type DeliveryAddress struct {
	addrType     AddressFlags
	nodeSelector uint32
	dataSelector uint64
}

func (v DeliveryAddress) IsZero() bool {
	return v.addrType == 0 && v.nodeSelector == 0
}

//func (v DeliveryAddress) String() string {
//	// TODO DeliveryAddress.String()
//}

type AddressFlags uint32

const DirectAddress AddressFlags = 0
const (
	RoleAddress AddressFlags = 1 << iota
)

type ReturnAddress struct {
	returnTo nwapi.Address
	returnID ShortShipmentID
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
