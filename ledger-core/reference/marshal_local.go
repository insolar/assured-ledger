package reference

import (
	"encoding/json"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func BinarySizeLocal(h LocalHolder) int {
	if h == nil || h.IsEmpty() {
		return 0
	}
	v := h.GetLocal()
	switch pn := v.GetPulseNumber(); {
	case pn.IsUnknown():
		return 0
	case pn.IsTimePulse() || pn == pulse.LocalRelative:
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
	case pn.IsTimePulse() || pn == pulse.LocalRelative:
		if len(data) >= LocalBinarySize {
			return WriteWholeLocalTo(v, data), nil
		}
	case len(data) >= LocalBinaryPulseAndScopeSize:
		v.pulseAndScopeToBytes(data)
		i := v.hashLen()
		if copy(data[LocalBinaryPulseAndScopeSize:], v.hash[:i]) == i {
			return LocalBinaryPulseAndScopeSize + i, nil
		}
	}
	return 0, throw.WithStackTop(io.ErrShortBuffer)
}

func MarshalLocalToSizedBuffer(h LocalHolder, data []byte) (int, error) {
	if h == nil || h.IsEmpty() {
		return 0, nil
	}

	v := h.GetLocal()
	switch pn := v.GetPulseNumber(); {
	case pn.IsUnknown():
		return 0, nil

	case pn.IsTimePulse() || pn == pulse.LocalRelative:
		if len(data) >= LocalBinarySize {
			return WriteWholeLocalTo(v, data[len(data)-LocalBinarySize:]), nil
		}
		return 0, throw.WithStackTop(io.ErrShortBuffer)

	case len(data) >= LocalBinaryPulseAndScopeSize:
		i := v.hashLen() + LocalBinaryPulseAndScopeSize
		if len(data) < i {
			return 0, throw.WithStackTop(io.ErrShortBuffer)
		}
		n := len(data) - i

		v.pulseAndScopeToBytes(data[n : n+LocalBinaryPulseAndScopeSize])
		copy(data[n+LocalBinaryPulseAndScopeSize:], v.hash[:i-LocalBinaryPulseAndScopeSize])
		return i, nil
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
	if err = func() error {
		switch n := len(data); {
		case n > LocalBinarySize:
			return throw.FailHere("too many bytes to unmarshal")
		case n < LocalBinaryPulseAndScopeSize:
			if n == 0 {
				return nil
			}
			return throw.FailHere("not enough bytes to unmarshal")
		default:
			v.pulseAndScope = pulseAndScopeFromBytes(data)

			switch pn := v.pulseAndScope.Pulse(); {
			case pn.IsTimePulse() || pn == pulse.LocalRelative:
				if len(data) != LocalBinarySize {
					return throw.FailHere("incorrect size of local hash")
				}
			case pn.IsSpecialOrPrivate():
				if len(data) > LocalBinaryPulseAndScopeSize && data[len(data)-1] == 0 {
					return throw.FailHere("incorrect padding of local hash")
				}
			case pn.IsUnknown():
				// this is a special case to represent base of record-ref
				if len(data) != LocalBinaryPulseAndScopeSize {
					return throw.FailHere("too many bytes to unmarshal zero ref")
				}
				return nil
			default:
				return throw.Impossible()
			}
			copy(v.hash[:], data[LocalBinaryPulseAndScopeSize:])
			return nil
		}
	}(); err != nil {
		return Local{}, err
	}
	return v, nil
}

func WriteWholeLocalTo(v Local, b []byte) int {
	if len(b) < LocalBinaryPulseAndScopeSize {
		return 0
	}
	v.pulseAndScopeToBytes(b)
	n := v.hash.CopyTo(b[LocalBinaryPulseAndScopeSize:])
	return LocalBinaryPulseAndScopeSize + n
}

func ReadWholeLocalFrom(b []byte) (v Local) {
	_ = b[LocalBinarySize-1]
	v.pulseAndScope = pulseAndScopeFromBytes(b)
	copy(v.hash[:], b[LocalBinaryPulseAndScopeSize:])
	return v
}
