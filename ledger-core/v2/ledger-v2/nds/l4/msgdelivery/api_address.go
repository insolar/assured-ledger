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
	return DeliveryAddress{addrType: directAddress, nodeSelector: uint32(id)}
}

func NewRoleAddress(roleId uint8, dataSelector uint64) DeliveryAddress {
	if roleId == 0 {
		panic(throw.IllegalValue())
	}
	return DeliveryAddress{addrType: roleAddress, nodeSelector: uint32(roleId), dataSelector: dataSelector}
}

type DeliveryAddress struct {
	addrType     addressFlags
	nodeSelector uint32
	dataSelector uint64
}

func (v DeliveryAddress) IsZero() bool {
	return v.addrType == 0 && v.nodeSelector == 0
}

//func (v DeliveryAddress) String() string {
//	// TODO DeliveryAddress.String()
//}

type addressFlags uint32

const directAddress addressFlags = 0
const (
	roleAddress addressFlags = 1 << iota
)

type DirectAddress = nwapi.ShortNodeID

type ReturnAddress struct {
	returnTo nwapi.Address
	returnId ShortShipmentID
}

func (v ReturnAddress) IsZero() bool {
	return v.returnTo.IsZero()
}

func (v ReturnAddress) IsValid() bool {
	return !v.returnTo.IsZero() && v.returnId != 0
}

func (v ReturnAddress) String() string {
	return v.returnTo.String() + "/" + strconv.FormatUint(uint64(v.returnId), 10)
}
