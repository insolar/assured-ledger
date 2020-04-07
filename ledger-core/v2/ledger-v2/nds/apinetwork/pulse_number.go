// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package apinetwork

import (
	"encoding/binary"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

func SerializePulseNumber(pn pulse.Number, w io.Writer) error {
	b := make([]byte, pulse.NumberSize)
	SerializePulseNumberToBytes(pn, b)
	_, err := w.Write(b)
	return err
}

func DeserializePulseNumber(r io.Reader) (pulse.Number, error) {
	b := make([]byte, pulse.NumberSize)
	if _, err := r.Read(b); err != nil {
		return 0, err
	}
	return DeserializePulseNumberFromBytes(b)
}

func SerializePulseNumberToBytes(pn pulse.Number, b []byte) {
	if !pulse.IsValidAsPulseNumber(int(pn)) {
		panic(throw.IllegalValue())
	}
	binary.LittleEndian.PutUint32(b, uint32(pn))
}

func DeserializePulseNumberFromBytes(b []byte) (pulse.Number, error) {
	if len(b) < pulse.NumberSize {
		return 0, throw.IllegalValue()
	}

	v := int(binary.LittleEndian.Uint32(b))
	if !pulse.IsValidAsPulseNumber(v) {
		return 0, throw.IllegalValue()
	}
	return pulse.OfInt(v), nil
}
