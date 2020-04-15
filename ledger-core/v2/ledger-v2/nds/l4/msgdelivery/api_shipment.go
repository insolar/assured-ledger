// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"encoding/binary"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/synckit"
)

type DeliveryAddress struct {
	addrType     DeliveryAddressFlags
	nodeSelector uint32
	dataSelector uint64
}

type DeliveryAddressFlags uint32

const directAddress DeliveryAddressFlags = 0
const (
	roleAddress DeliveryAddressFlags = 1 << iota
)

type DirectAddress = nwapi.ShortNodeID

type ReturnAddress struct {
	returnTo DirectAddress
	returnId ShipmentID
}

type PulseTTL struct {
	RefPulse pulse.Number
	RefCount uint32
	TTL      uint8
}

type Shipment struct {
	Head   nwapi.SizeAwareSerializer
	Body   nwapi.SizeAwareSerializer
	Cancel *synckit.ChainedCancel
	PN     pulse.Number
	// TTL defines how many pulses this shipment can survive before cancellation
	TTL      uint8
	Policies DeliveryPolicies
}

func AsShipmentID(node uint32, id ShortShipmentID) ShipmentID {
	if id == 0 {
		return 0
	}
	return ShipmentID(node)<<32 | ShipmentID(id)
}

type ShipmentID uint64 // NodeId + ShortShipmentID

func (v ShipmentID) ShortId() ShortShipmentID {
	return ShortShipmentID(v)
}

type ShortShipmentID uint32

const ShortShipmentIDByteSize = 4

func (v ShortShipmentID) SimpleWriteTo(writer io.Writer) error {
	var b [ShortShipmentIDByteSize]byte
	v.PutTo(b[:])
	switch n, err := writer.Write(b[:]); {
	case err != nil:
		return err
	case n != len(b):
		return io.ErrShortWrite
	default:
		return nil
	}
}

func (v ShortShipmentID) PutTo(b []byte) int {
	binary.LittleEndian.PutUint64(b, uint64(v))
	return ShortShipmentIDByteSize
}

func ShortShipmentIDReadFrom(reader io.Reader) (ShortShipmentID, error) {
	b := make([]byte, ShortShipmentIDByteSize)
	if _, err := io.ReadFull(reader, b); err != nil {
		return 0, err
	}
	return ShortShipmentIDReadFromBytes(b), nil
}

func ShortShipmentIDReadFromBytes(b []byte) ShortShipmentID {
	return ShortShipmentID(binary.LittleEndian.Uint32(b))
}
