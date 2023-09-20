package rmsbox

import (
	"bytes"
	"encoding"
	"fmt"
	"io"
	"reflect"

	"github.com/gogo/protobuf/proto"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewRawBytes(value []byte) (rb RawBinary) {
	rb.SetBytes(value)
	return
}

func NewRaw(value longbits.FixedReader) (rb RawBinary) {
	rb.Set(value)
	return
}

func NewRawMarshal(value proto.ProtoSizer) (rb RawBinary) {
	rb.SetProto(value)
	return
}

func NewRawMarshalWithSizer(value proto.Sizer) (rb RawBinary) {
	rb.SetProtoWithSizer(value)
	return
}

var _ longbits.FixedReader = &RawBinary{}

type RawBinary struct {
	value interface{}
}

func (p RawBinary) AsBinary() Binary {
	return Binary{p}
}

func (p RawBinary) IsZero() bool {
	return p.value == nil
}

func (p *RawBinary) IsEmpty() bool {
	return p._size() == 0
}

func (p *RawBinary) _serializableSize() int {
	switch vv := p.value.(type) {
	case proto.ProtoSizer:
		return vv.ProtoSize()
	case proto.Sizer:
		return vv.Size()
	default:
		return -1
	}
}

func (p *RawBinary) _size() int {
	if p == nil {
		return 0
	}
	switch vv := p.value.(type) {
	case nil:
		return 0
	case longbits.FixedReader:
		return vv.FixedByteSize()
	default:
		return p._serializableSize()
	}
}

func (p *RawBinary) MarshalText() ([]byte, error) {
	switch vv := p.value.(type) {
	case nil:
		return nil, nil
	case encoding.TextMarshaler:
		return vv.MarshalText()
	case fmt.Stringer:
		return []byte(vv.String()), nil
	default:
		return []byte(fmt.Sprintf("%T", vv)), nil
	}
}

func (p *RawBinary) _marshalTo(b []byte) (int, error) {
	switch vv := p.value.(type) {
	case interface{ MarshalTo(b []byte) (int, error) }:
		return vv.MarshalTo(b)
	case interface{ Marshal() ([]byte, error) }:
		bb, err := vv.Marshal()
		if err != nil {
			return 0, err
		}
		if len(bb) > len(b) {
			return 0, io.ErrShortBuffer
		}
		return copy(b, bb), nil
	default:
		panic(throw.Unsupported())
	}
}

/*
	ATTENTION! These methods of *RawBinary are NOT-exported intentionally,
	because RawBinary should not be directly usable for serialization.

*/

func (p *RawBinary) protoSize() int {
	n := p._size()
	if n < 0 {
		panic(throw.Unsupported())
	}
	return n
}

func (p *RawBinary) marshalTo(b []byte) (int, error) {
	switch vv := p.value.(type) {
	case nil:
		return 0, nil
	case longbits.FixedReader:
		if vv.FixedByteSize() > len(b) {
			return 0, io.ErrShortBuffer
		}
		return vv.CopyTo(b), nil
	default:
		return p._marshalTo(b)
	}
}

func (p *RawBinary) marshalToSizedBuffer(b []byte) (n int, err error) {
	expected := 0
	switch vv := p.value.(type) {
	case nil:
		return 0, nil
	case longbits.FixedReader:
		expected = vv.FixedByteSize()
		if expected > 0 {
			n = vv.CopyTo(b[len(b)-expected:])
		}
	case interface{ BinaryMarshalToSizedBuffer(b []byte) (int, error) }:
		return vv.BinaryMarshalToSizedBuffer(b)
	case interface{ MarshalTo(b []byte) (int, error) }:
		n = p._serializableSize()
		switch {
		case n > 0:
			return vv.MarshalTo(b[len(b)-n:])
		case n == 0:
			return 0, nil
		default:
			n, err = vv.MarshalTo(b)
			if err != nil {
				return 0, err
			}
			if n < len(b) {
				copy(b[len(b)-n:], b)
			}
			return n, nil
		}
	case interface{ Marshal() ([]byte, error) }:
		bb, err := vv.Marshal()
		if err != nil {
			return 0, err
		}
		expected = len(bb)
		if expected <= len(b) {
			n = copy(b[len(b)-expected:], bb)
		}
	default:
		panic(throw.Unsupported())
	}

	if expected != n {
		return 0, io.ErrShortBuffer
	}
	return n, nil
}

func (p *RawBinary) unmarshal(b []byte) error {
	p.value = longbits.CopyBytes(b)
	return nil
}

func (p *RawBinary) SetBytes(value []byte) {
	if len(value) == 0 {
		p.value = nil
		return
	}
	p.value = longbits.WrapBytes(value)
}

func (p *RawBinary) Set(value longbits.FixedReader) {
	p.value = value
}

func (p *RawBinary) SetProto(value proto.ProtoSizer) {
	p._setProto(value)
}

func (p *RawBinary) SetProtoWithSizer(value proto.Sizer) {
	p._setProto(value)
}

func (p *RawBinary) _setProto(value interface{}) {
	switch p.value.(type) {
	case nil:
		//
	case interface{ MarshalTo(b []byte) (int, error) }:
		//
	case interface{ Marshal() ([]byte, error) }:
		//
	default:
		panic(throw.IllegalValue())
	}
	p.value = value
}

func (p *RawBinary) GetBytes() []byte {
	switch vv := p.value.(type) {
	case nil:
		return nil
	case longbits.FixedReader:
		b := longbits.AsBytes(vv)
		if len(b) == 0 {
			return nil
		}
		return b

	case interface{ Marshal() ([]byte, error) }:
		switch b, err := vv.Marshal(); {
		case err != nil:
			panic(throw.WithStackTop(err))
		case len(b) == 0:
			return nil
		default:
			return b
		}
	case interface{ MarshalTo(b []byte) (int, error) }:
		b := make([]byte, p._serializableSize())

		switch n, err := vv.MarshalTo(b); {
		case err != nil:
			panic(throw.WithStackTop(err))
		case n == 0:
			return nil
		default:
			return b[:n]
		}
	default:
		panic(throw.Unsupported())
	}
}

func (p *RawBinary) EqualRaw(o RawBinary) bool {
	return p._equal(&o)
}

func (p *RawBinary) Equal(o longbits.FixedReader) bool {
	switch {
	case p == o:
		return true
	case p == nil:
		return o.FixedByteSize() == 0
	case o == nil:
		return p.FixedByteSize() == 0
	}
	if ov, ok := o.(*RawBinary); ok {
		return p._equal(ov)
	}
	return p._equalReader(o)
}

func (p *RawBinary) _equal(o *RawBinary) bool {
	switch n := p._size(); {
	case n != o._size():
		return false
	case n == 0:
		return true
	}

	switch vv := p.value.(type) {
	case longbits.FixedReader:
		if ov, ok := o.value.(longbits.FixedReader); ok {
			return longbits.Equal(ov, vv)
		}
		return longbits.EqualToBytes(vv, o.GetBytes())

	case interface{ Equal(interface{}) bool }:
		if reflect.TypeOf(p.value) == reflect.TypeOf(o.value) {
			if ov, ok := o.value.(interface{ Equal(interface{}) bool }); ok {
				return vv.Equal(ov)
			}
		}
	}

	return bytes.Equal(p.GetBytes(), o.GetBytes())
}

func (p *RawBinary) _equalReader(o longbits.FixedReader) bool {
	if vv, ok := p.value.(longbits.FixedReader); ok {
		return longbits.Equal(o, vv)
	}
	return longbits.EqualToBytes(o, p.GetBytes())
}

func (p *RawBinary) FixedByteSize() int {
	return p._size()
}

func (p *RawBinary) WriteTo(w io.Writer) (int64, error) {
	switch vv := p.value.(type) {
	case nil:
		return 0, nil
	case io.WriterTo: // works for fixed reader
		return vv.WriteTo(w)
	default:
		n, err := w.Write(p.GetBytes())
		return int64(n), err
	}
}

func (p *RawBinary) CopyTo(b []byte) int {
	switch vv := p.value.(type) {
	case nil:
		return 0
	case longbits.FixedReader: // works for fixed reader
		return vv.CopyTo(b)
	default:
		n, err := p._marshalTo(b)
		if err != nil {
			panic(throw.WithStackTop(err))
		}
		return n
	}
}

func (p *RawBinary) AsByteString() longbits.ByteString {
	switch vv := p.value.(type) {
	case nil:
		return ""
	case longbits.FixedReader: // works for fixed reader
		return vv.AsByteString()
	default:
		return longbits.CopyBytes(p.GetBytes())
	}
}
