// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reference

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

func ProtoSizeLocal(h LocalHolder) int {
	if h == nil || h.IsEmpty() {
		return 0
	}
	v := h.GetLocal()
	switch pn := v.GetPulseNumber(); {
	case pn.IsUnknown():
		return 0
	case pn.IsTimePulse():
		return LocalBinarySize
	}
	return LocalBinaryPulseAndScopeSize + v.hashLen()
}

func MarshalLocal(h LocalHolder) ([]byte, error) {
	if h == nil || h.IsEmpty() {
		return nil, nil
	}
	b := make([]byte, LocalBinarySize)
	n, err := MarshalLocalTo(h, b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}

func MarshalLocalTo(h LocalHolder, data []byte) (int, error) {
	if h == nil || h.IsEmpty() {
		return 0, nil
	}
	v := h.GetLocal()
	switch pn := v.GetPulseNumber(); {
	case pn.IsUnknown():
		return 0, nil
	case pn.IsTimePulse():
		if len(data) >= LocalBinarySize {
			return WriteWholeLocalTo(v, data), nil
		}
	case len(data) >= LocalBinaryPulseAndScopeSize:
		copy(data, v.pulseAndScopeAsBytes())
		i := v.hashLen()
		if copy(data[LocalBinaryPulseAndScopeSize:], v.hash[:i]) == i {
			return LocalBinaryPulseAndScopeSize + i, nil
		}
	}
	return 0, throw.WithStackTop(io.ErrShortBuffer)
}

func UnmarshalLocalJSON(data []byte) (v Local, err error) {
	var repr interface{}

	err = json.Unmarshal(data, &repr)
	if err != nil {
		err = throw.W(err, "failed to unmarshal reference.Local")
		return
	}

	switch realRepr := repr.(type) {
	case string:
		var decoded Global
		decoded, err = DefaultDecoder().Decode(realRepr)
		if err != nil {
			err = throw.W(err, "failed to unmarshal reference.Local")
			return
		}
		// TODO must check and fail if it was a full reference
		v = decoded.GetLocal()
	case nil:
		//
	default:
		err = throw.W(err, "unexpected type %T when unmarshal reference.Local", repr)
	}
	return
}

func MarshalLocalJSON(v LocalHolder) (b []byte, err error) {
	if v == nil {
		return json.Marshal(nil)
	}
	return json.Marshal(v.GetLocal().Encode(DefaultEncoder()))
}

func UnmarshalLocal(data []byte) (v Local, err error) {
	switch n := len(data); {
	case n > LocalBinarySize:
		return Local{}, errors.New("too many bytes to unmarshal")
	case n < LocalBinaryPulseAndScopeSize:
		return Local{}, errors.New("not enough bytes to unmarshal")
	default:
		writer := v.asWriter()
		for i := 0; i < n; i++ {
			_ = writer.WriteByte(data[i])
		}
		// TODO do strick check that with Pulse().IsTimePulse() size == LocalBinarySize, and there are no tailing zeros otherwise
		for i := n; i < LocalBinarySize; i++ {
			_ = writer.WriteByte(0)
		}
		return
	}
}

func WriteWholeLocalTo(v Local, b []byte) int {
	if len(b) < LocalBinaryPulseAndScopeSize {
		return copy(b, v.pulseAndScopeAsBytes())
	}

	byteOrder.PutUint32(b, v.pulseAndScope)
	n := v.hash.CopyTo(b[LocalBinaryPulseAndScopeSize:])
	return LocalBinaryPulseAndScopeSize + n
}

func ReadWholeLocalFrom(b []byte) (v Local) {
	writer := v.asWriter()
	for i := 0; i < LocalBinarySize; i++ {
		_ = writer.WriteByte(b[i])
	}
	return v
}
