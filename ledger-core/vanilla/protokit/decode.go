package protokit

import (
	"errors"
	"fmt"
	"io"
)

var errOverflow = errors.New("proto: uint32 overflow")

func DecodeVarint(r io.ByteReader) (uint64, error) {
	b, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	return decodeVarint(b, r)
}

// Continues to read Varint that was stated with the given (b)
// See also binary.ReadUvarint(r) here
func decodeVarint(b byte, r io.ByteReader) (n uint64, err error) {
	v := uint64(b & 0x7F)

	for i := uint8(7); i < 64; i += 7 {
		if b&0x80 == 0 {
			return v, nil
		}
		if b, err = r.ReadByte(); err != nil {
			return 0, err
		}
		v |= uint64(b&0x7F) << i
	}

	if b > 1 {
		return 0, errOverflow
	}
	return v, nil
}

func DecodeFixed64(r io.ByteReader) (uint64, error) {
	if v, err := DecodeFixed32(r); err != nil {
		return 0, err
	} else if v2, err := DecodeFixed32(r); err != nil {
		return 0, err
	} else {
		return v2<<32 | v, nil
	}
}

func DecodeFixed32(r io.ByteReader) (v uint64, err error) {
	// NB! uint64 result is NOT a mistake
	var b byte
	if b, err = r.ReadByte(); err != nil {
		return 0, err
	}
	v = uint64(b)
	if b, err = r.ReadByte(); err != nil {
		return 0, err
	}
	v |= uint64(b) << 8
	if b, err = r.ReadByte(); err != nil {
		return 0, err
	}
	v |= uint64(b) << 16
	if b, err = r.ReadByte(); err != nil {
		return 0, err
	}
	v |= uint64(b) << 24
	return v, nil
}

func DecodeFixed64FromBytes(b []byte) (uint64, int) {
	_ = b[7]
	v := uint64(b[0])
	v |= uint64(b[1]) << 8
	v |= uint64(b[2]) << 16
	v |= uint64(b[3]) << 24
	v |= uint64(b[4]) << 32
	v |= uint64(b[5]) << 40
	v |= uint64(b[6]) << 48
	v |= uint64(b[7]) << 56

	return v, 8
}

func DecodeFixed64FromBytesWithError(b []byte) (v uint64, n int, err error) {
	if len(b) < 8 {
		return 0, 0, io.ErrUnexpectedEOF
	}
	v, n = DecodeFixed64FromBytes(b)
	return
}

func DecodeFixed32FromBytes(b []byte) (uint64, int) {
	// NB! uint64 result is NOT a mistake
	_ = b[3]
	v := uint64(b[0])
	v |= uint64(b[1]) << 8
	v |= uint64(b[2]) << 16
	v |= uint64(b[3]) << 24

	return v, 4
}

func DecodeFixed32FromBytesWithError(b []byte) (v uint64, n int, err error) {
	// NB! uint64 result is NOT a mistake
	if len(b) < 4 {
		return 0, 0, io.ErrUnexpectedEOF
	}
	v, n = DecodeFixed32FromBytes(b)
	return
}

func DecodeVarintFromBytes(bb []byte) (u uint64, n int) {
	b := bb[0]

	v := uint64(b & 0x7F)

	for i := uint8(7); i < 64; i += 7 {
		n++
		if b&0x80 == 0 {
			return v, n
		}
		b = bb[n]
		v |= uint64(b&0x7F) << i
	}

	if b > 1 {
		return 0, 0 // errOverflow
	}
	n++
	return v, n
}

func DecodeVarintFromBytesWithError(bb []byte) (u uint64, n int, err error) {
	if len(bb) == 0 {
		return 0, 0, io.ErrUnexpectedEOF
	}
	b := bb[0]

	v := uint64(b & 0x7F)

	for i := uint8(7); i < 64; i += 7 {
		n++
		if b&0x80 == 0 {
			return v, n, nil
		}
		if len(bb) <= n {
			return 0, 0, io.ErrUnexpectedEOF
		}
		b = bb[n]
		v |= uint64(b&0x7F) << i
	}

	if b > 1 {
		return 0, 0, errOverflow
	}
	n++
	return v, n, nil
}

func IsValidFirstByteOfTag(firstByte byte) bool {
	return firstByte > maskWireType && WireType(firstByte&maskWireType).IsValid()
}

func (v WireType) IsValidFirstByte(firstByte byte) bool {
	return firstByte > maskWireType && firstByte&maskWireType == byte(v) && v.IsValid()
}

func (v WireTag) IsValidFirstByte(firstByte byte) bool {
	switch {
	case !IsValidFirstByteOfTag(firstByte):
		return false
	case v <= 0xFF:
		return firstByte == byte(v)
	default:
		return firstByte == byte(v)|0x80
	}
}

func _readTag(firstByte byte, r io.ByteReader) (wt WireTag, err error) {
	var x uint64
	x, err = decodeVarint(firstByte, r)
	if err != nil {
		return 0, err
	}
	return SafeWireTag(x)
}

func TryReadAnyTag(r io.ByteScanner) (wt WireTag, err error) {
	var b byte
	b, err = r.ReadByte()
	switch {
	case err != nil:
		return 0, err
	case !IsValidFirstByteOfTag(b):
		return 0, r.UnreadByte()
	}
	return _readTag(b, r)
}

func MustReadAnyTag(r io.ByteReader) (wt WireTag, err error) {
	var b byte
	b, err = r.ReadByte()
	switch {
	case err != nil:
		return 0, err
	case !IsValidFirstByteOfTag(b):
		return 0, fmt.Errorf("invalid wire tag, wrong first byte: %x", b)
	}
	return _readTag(b, r)
}

func MustReadAnyTagValue(r io.ByteReader) (rt WireTag, u uint64, err error) {
	rt, err = MustReadAnyTag(r)
	if err != nil {
		return 0, 0, err
	}
	u, err = rt.ReadValue(r)
	return
}

func TryReadAnyTagValue(r io.ByteScanner) (rt WireTag, u uint64, err error) {
	rt, err = TryReadAnyTag(r)
	if err != nil || rt.IsZero() {
		return 0, 0, err
	}
	u, err = rt.ReadValue(r)
	return
}

func (v WireTag) TryReadTag(r io.ByteScanner) (WireTag, error) {
	b, err := r.ReadByte()
	switch {
	case err != nil:
		return 0, err
	case !v.IsValidFirstByte(b):
		return 0, r.UnreadByte()
	}
	var rt WireTag
	rt, err = _readTag(b, r)
	if err != nil {
		return 0, err
	}
	return rt, rt.CheckTag(v)
}

func (v WireType) TryReadTag(r io.ByteScanner) (WireTag, error) {
	b, err := r.ReadByte()
	switch {
	case err != nil:
		return 0, err
	case !v.IsValidFirstByte(b):
		return 0, r.UnreadByte()
	}
	var rt WireTag
	rt, err = _readTag(b, r)
	if err != nil {
		return 0, err
	}
	return rt, rt.CheckType(v)
}

func (v WireTag) ReadTag(r io.ByteReader) error {
	rt, err := MustReadAnyTag(r)
	if err != nil {
		return err
	}
	return rt.CheckTag(v)
}

func (v WireType) ReadTag(r io.ByteReader) (WireTag, error) {
	rt, err := MustReadAnyTag(r)
	if err != nil {
		return rt, err
	}
	return rt, rt.CheckType(v)
}

func (v WireTag) ReadTagValue(r io.ByteReader) (uint64, error) {
	err := v.ReadTag(r)
	if err != nil {
		return 0, err
	}
	return v.ReadValue(r)
}

func (v WireType) ReadTagValue(r io.ByteReader) (WireTag, uint64, error) {
	rt, err := v.ReadTag(r)
	if err != nil {
		return rt, 0, err
	}
	var u uint64
	u, err = rt.ReadValue(r)
	return rt, u, err
}
