// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l4

import (
	"encoding/binary"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/apinetwork"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/synckit"
)

type DeliveryAddress struct {
}

type ReturnAddress struct {
	returnTo DeliveryAddress
	returnId ParcelId
}

type DeliveryParcel struct {
	Head   apinetwork.SizeAwareSerializer
	Body   apinetwork.SizeAwareSerializer
	Cancel *synckit.ChainedCancel
	// TTL defines how many pulses this parcel can survive before cancellation
	TTL      uint8
	Policies DeliveryPolicies
}

type ParcelId uint64

const ParcelIdByteSize = 8

func (v ParcelId) WriteTo(writer io.Writer) error {
	var b [ParcelIdByteSize]byte
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

func (v ParcelId) PutTo(b []byte) int {
	binary.LittleEndian.PutUint64(b, uint64(v))
	return ParcelIdByteSize
}

func ParcelIdReadFrom(reader io.Reader) (ParcelId, error) {
	b := make([]byte, ParcelIdByteSize)
	if _, err := io.ReadFull(reader, b); err != nil {
		return 0, err
	}
	return ParcelIdReadFromBytes(b), nil
}

func ParcelIdReadFromBytes(b []byte) ParcelId {
	return ParcelId(binary.LittleEndian.Uint64(b))
}
