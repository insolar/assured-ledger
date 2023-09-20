package reference

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func BinarySize(h Holder) int {
	if h == nil {
		return 0
	}
	base := h.GetBase()
	local := h.GetLocal()

	switch {
	case base.IsEmpty():
		if local.IsEmpty() {
			return 0
		}
		// explicit form for record-ref
		return LocalBinarySize + LocalBinaryPulseAndScopeSize
	case base == local:
		// compact form for self-ref
		return BinarySizeLocal(local)
	}
	return LocalBinarySize + BinarySizeLocal(base)
}

func MarshalTo(h Holder, b []byte) (int, error) {
	if h == nil {
		return 0, nil
	}
	base := h.GetBase()
	local := h.GetLocal()

	switch {
	case base.IsEmpty():
		if local.IsEmpty() {
			return 0, nil
		}
	case base == local:
		// compact form for self-ref
		return MarshalLocalTo(local, b)
	}
	return _marshal(base, local, b)
}

func _marshal(base, local Local, b []byte) (int, error) {
	n := WriteWholeLocalTo(local, b)
	switch {
	case n != LocalBinarySize:
		return 0, throw.WithStackTop(io.ErrShortBuffer)
	case !base.IsEmpty():
		//
	case len(b) < LocalBinarySize+LocalBinaryPulseAndScopeSize:
		return 0, throw.WithStackTop(io.ErrShortBuffer)
	default:
		// explicit form for record-ref
		zeros := [LocalBinaryPulseAndScopeSize]byte{}
		copy(b[n:], zeros[:])
		return LocalBinarySize + LocalBinaryPulseAndScopeSize, nil
	}

	if n2, err := MarshalLocalTo(base, b[n:]); err == nil {
		return n + n2, err
	}
	return 0, throw.WithStackTop(io.ErrShortBuffer)
}

func Marshal(h Holder) ([]byte, error) {
	if h == nil {
		return nil, nil
	}

	base := h.GetBase()
	local := h.GetLocal()

	switch {
	case base.IsEmpty():
		if local.IsEmpty() {
			return nil, nil
		}

		// explicit form for record-ref
		b := make([]byte, LocalBinarySize+LocalBinaryPulseAndScopeSize)
		n := WriteWholeLocalTo(local, b)
		if n != LocalBinarySize {
			return nil, throw.WithStackTop(io.ErrShortBuffer)
		}
		// don't need to zero-out last bytes
		return b, nil

	case base == local:
		// compact form for self-ref
		return MarshalLocal(local)
	}

	b := make([]byte, LocalBinarySize<<1)
	n, err := _marshal(base, local, b)
	if err != nil {
		// must not happen
		panic(throw.W(err, "impossible"))
	}
	return b[:n], err
}

func MarshalToSizedBuffer(h Holder, b []byte) (int, error) {
	if h == nil {
		return 0, nil
	}
	base := h.GetBase()
	local := h.GetLocal()

	switch {
	case base.IsEmpty():
		if local.IsEmpty() {
			return 0, nil
		}
		n := LocalBinarySize + LocalBinaryPulseAndScopeSize
		n2, err := _marshal(base, local, b[len(b)-n:])
		if err != nil || n2 != n {
			// must not happen
			panic(throw.W(err, "impossible"))
		}
		return n, nil

	case local == base:
		return MarshalLocalToSizedBuffer(local, b)
	}

	n, err := MarshalLocalToSizedBuffer(base, b)
	if err != nil {
		return 0, err
	}

	n += LocalBinarySize
	if len(b) < n {
		return 0, throw.WithStackTop(io.ErrShortBuffer)
	}

	WriteWholeLocalTo(local, b[len(b)-n:])
	return n, nil
}

func Encode(h Holder) string {
	s, err := DefaultEncoder().Encode(h)
	if err != nil {
		panic(throw.WithStackTop(err))
	}
	return s
}

func EncodeLocal(h LocalHolder) string {
	s, err := DefaultEncoder().EncodeRecord(h)
	if err != nil {
		panic(throw.WithStackTop(err))
	}
	return s
}

func Decode(s string) (Holder, error) {
	return DefaultDecoder().DecodeHolder(s)
}

func MarshalJSON(h Holder) ([]byte, error) {
	if h == nil {
		return json.Marshal(nil)
	}
	s, err := DefaultEncoder().Encode(h)
	if err != nil {
		return nil, err
	}
	return json.Marshal(s)
}

func UnmarshalJSON(b []byte) (v Global, err error) {
	err = _unmarshalJSON(&v, b, DefaultDecoder())
	return
}

func _unmarshalJSON(v *Global, data []byte, decoder GlobalDecoder) error {
	var repr interface{}

	err := json.Unmarshal(data, &repr)
	if err != nil {
		return throw.W(err, "failed to unmarshal reference.Global")
	}

	switch realRepr := repr.(type) {
	case string:
		decoded, err := decoder.DecodeHolder(realRepr)
		if err != nil {
			return throw.W(err, "failed to unmarshal reference.Global")
		}

		v.addressLocal = decoded.GetLocal()
		v.addressBase = decoded.GetBase()
	case nil:
	default:
		return throw.W(err, fmt.Sprintf("unexpected type %T when unmarshal reference.Global", repr))
	}

	return nil
}

func Unmarshal(b []byte) (base, local Local, err error) {
	var v Global
	v, err = UnmarshalGlobal(b)
	return v.addressBase, v.addressLocal, err
}

func UnmarshalGlobal(b []byte) (v Global, err error) {
	switch n := len(b); {
	case n == 0:
		return
	case n < LocalBinaryPulseAndScopeSize:
		err = throw.WithStackTop(io.ErrShortBuffer)
		return
	case n <= LocalBinarySize:
		// self-ref
		v.addressLocal, err = UnmarshalLocal(b)
		v.addressBase = v.addressLocal
		return
	case n > GlobalBinarySize:
		err = throw.FailHere("too much bytes to unmarshal reference.Global")
		return
	default:
		v.addressLocal = ReadWholeLocalFrom(b[:LocalBinarySize])
		v.addressBase, err = UnmarshalLocal(b[LocalBinarySize:])
		if err != nil {
			return Global{}, err
		}
		return
	}
}
