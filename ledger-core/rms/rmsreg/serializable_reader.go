package rmsreg

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewSerializableFromReader(s io.Reader, dataSize int) SerializableFromReader {
	switch {
	case dataSize < 0:
		panic(throw.IllegalValue())
	case dataSize == 0:
		s = nil
	}
	return SerializableFromReader{source: s, size: dataSize}
}

var _ Serializable = &SerializableFromReader{}

type SerializableFromReader struct {
	size       int
	cachedData []byte
	source     io.Reader
}

func (p *SerializableFromReader) ProtoSize() int {
	if p.size < 0 {
		panic(throw.IllegalState())
	}
	return p.size
}

func (p *SerializableFromReader) Marshal() (b []byte, err error) {
	switch {
	case p.size == 0:
		return nil, nil
	case p.size < 0:
		panic(throw.IllegalState())
	case p.cachedData != nil:
		//
	case p.source == nil:
		panic(throw.IllegalState())
	}

	if b, err = p._marshalTo(nil); err != nil {
		b = nil
	}
	return b, err
}

func (p *SerializableFromReader) MarshalTo(b []byte) (n int, err error) {
	switch {
	case p.size == 0:
		return 0, nil
	case p.size < 0:
		panic(throw.IllegalState())
	case p.cachedData != nil:
		//
	case p.source != nil:
		panic(throw.IllegalState())
	}

	b, err = p._marshalTo(b)
	return len(b), err
}

func (p *SerializableFromReader) _marshalTo(bb []byte) (b []byte, err error) {
	b = bb

	switch {
	case p.cachedData == nil:
		switch {
		case b == nil:
			b = make([]byte, p.size)
		case len(b) < p.size:
			return nil, io.ErrShortBuffer
		default:
			b = b[:p.size]
		}
	case b == nil:
		return append([]byte(nil), p.cachedData...), nil
	case len(b) < len(p.cachedData):
		return nil, io.ErrShortBuffer
	default:
		return b[:copy(b, p.cachedData)], nil
	}

	n := 0
	if at, ok := p.source.(io.ReaderAt); ok {
		n, err = at.ReadAt(b, 0)
		if n == p.size {
			return b, nil
		}
	} else {
		buf := make([]byte, p.size)
		n, err = io.ReadFull(p.source, buf)
		p.source = nil
		if n == p.size {
			p.cachedData = buf
			return b[:copy(b, buf)], nil
		}
	}

	p.size = -1
	p.cachedData = nil
	if err == io.EOF || err == nil {
		err = io.ErrUnexpectedEOF
	}
	return nil, err
}

func (p *SerializableFromReader) Unmarshal([]byte) error {
	panic(throw.Unsupported())
}
