// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"encoding/binary"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/apinetwork"
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

type DirectAddress = apinetwork.ShortNodeID

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
	Head   apinetwork.SizeAwareSerializer
	Body   apinetwork.SizeAwareSerializer
	Cancel *synckit.ChainedCancel
	PN     pulse.Number
	// TTL defines how many pulses this shipment can survive before cancellation
	TTL      uint8
	Policies DeliveryPolicies
}

type ShortShipmentId uint32
type ShipmentID uint64 // NodeId + ShortShipmentId

const ShipmentIdByteSize = 8

func (v ShipmentID) ShortId() ShortShipmentId {
	return ShortShipmentId(v)
}

func (v ShipmentID) WriteTo(writer io.Writer) error {
	var b [ShipmentIdByteSize]byte
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

func (v ShipmentID) PutTo(b []byte) int {
	binary.LittleEndian.PutUint64(b, uint64(v))
	return ShipmentIdByteSize
}

func ShipmentIdReadFrom(reader io.Reader) (ShipmentID, error) {
	b := make([]byte, ShipmentIdByteSize)
	if _, err := io.ReadFull(reader, b); err != nil {
		return 0, err
	}
	return ShipmentIdReadFromBytes(b), nil
}

func ShipmentIdReadFromBytes(b []byte) ShipmentID {
	return ShipmentID(binary.LittleEndian.Uint64(b))
}
