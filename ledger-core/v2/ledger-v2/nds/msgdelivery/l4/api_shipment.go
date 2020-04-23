// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l4

import (
	"encoding/binary"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/synckit"
)

func AsShipmentID(node uint32, id ShortShipmentID) ShipmentID {
	if id == 0 {
		return 0
	}
	return ShipmentID(node)<<32 | ShipmentID(id)
}

type ShipmentID uint64 // NodeId + ShortShipmentID

func (v ShipmentID) NodeID() uint32 {
	return uint32(v >> 32)
}

func (v ShipmentID) ShortID() ShortShipmentID {
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
	binary.LittleEndian.PutUint32(b, uint32(v))
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

type Shipment struct {
	Head   nwapi.SizeAwareSerializer
	Body   nwapi.SizeAwareSerializer
	Cancel *synckit.ChainedCancel
	PN     pulse.Number
	// TTL defines how many pulses this shipment can survive before cancellation
	TTL      uint8
	Policies DeliveryPolicies
}

type ShipmentRequest struct {
	ReceiveFn ReceiverFunc
	Cancel    *synckit.ChainedCancel
}
