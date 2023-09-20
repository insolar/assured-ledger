package longbits

import (
	"io"
	"math/bits"
)

type bitReader struct {
	accumulator byte
	accInit     byte
	accBit      byte
}

func (p *bitReader) _rightShift() bool {
	switch p.accInit {
	case initLSB:
		return false
	case initMSB:
		return true
	default:
		panic("illegal state")
	}
}

func (p *bitReader) align() (rightShift bool, ofs uint8) {
	if p._rightShift() {
		if p.accBit == 0 {
			return true, 0
		}
		return true, uint8(bits.LeadingZeros8(p.accBit))
	}
	if p.accBit == 0 {
		return false, 0
	}
	return false, uint8(bits.TrailingZeros8(p.accBit))
}

func (p *bitReader) AlignOffset() uint8 {
	_, ofs := p.align()
	return ofs
}

func (p *bitReader) ensure(readFn func() (byte, error)) error {
	if p.accBit != 0 {
		return nil
	}
	if p.accInit == 0 {
		p.accInit = initLSB
	}
	b, err := readFn()
	if err != nil {
		return err
	}
	p.accumulator = b
	p.accBit = p.accInit
	return nil
}

func (p *bitReader) readNext(readFn func() (byte, error)) (int, error) {
	if e := p.ensure(readFn); e != nil {
		return 0, e
	}

	m := p.accBit
	if rightShift, _ := p.align(); rightShift {
		p.accBit >>= 1
	} else {
		p.accBit <<= 1
	}
	return int(p.accumulator & m), nil
}

func (p *bitReader) readByte(readFn func() (byte, error)) (byte, error) {
	switch p.accBit {
	case 0:
		if p.accInit == 0 {
			p.accInit = initLSB
		}
		if p._rightShift() {
			v, e := readFn()
			return bits.Reverse8(v), e
		}
		return readFn()
	case p.accInit:
		p.accBit = 0
		if p._rightShift() {
			return bits.Reverse8(p.accumulator), nil
		}
		return p.accumulator, nil
	}

	v := p.accumulator
	b, err := readFn()
	if err != nil {
		return 0, err
	}
	p.accumulator = b

	w := p.accumulator
	if rightShift, usedBits := p.align(); rightShift {
		v <<= usedBits
		w >>= 8 - usedBits
		v = bits.Reverse8(v | w)
	} else {
		v >>= usedBits
		w <<= 8 - usedBits
		v |= w
	}

	return v, nil
}

func (p *bitReader) readSubByte(bitLen uint8, readFn func() (byte, error)) (uint8, error) {
	switch {
	case bitLen == 0:
		return 0, nil
	case bitLen == 1:
		switch v, e := p.readNext(readFn); {
		case e != nil:
			return 0, e
		case v != 0:
			return 1, nil
		default:
			return 0, nil
		}
	case bitLen == 8:
		return p.readByte(readFn)
	case bitLen > 8:
		panic("illegal value")
	}

	if e := p.ensure(readFn); e != nil {
		return 0, e
	}

	rightShift, usedBits := p.align()
	remainBits := 8 - usedBits

	if bitLen <= remainBits {
		if rightShift {
			p.accBit >>= bitLen
			v := p.accumulator
			if p.accBit != 0 {
				v &^= (p.accBit << 1) - 1
			}
			return bits.Reverse8(v << usedBits), nil
		}

		p.accBit <<= bitLen
		v := p.accumulator & (p.accBit - 1)
		return v >> usedBits, nil
	}
	bitLen -= remainBits

	v := p.accumulator
	p.accBit = 0
	if e := p.ensure(readFn); e != nil {
		return 0, e
	}

	if rightShift {
		p.accBit >>= bitLen
		v &= 0xFF >> usedBits
		w := uint16(v)<<8 | uint16(p.accumulator)
		w <<= bitLen
		v = uint8(w >> 8)
		v <<= usedBits - bitLen
		v = bits.Reverse8(v)
	} else {
		p.accBit <<= bitLen
		v &= 0xFF << usedBits
		w := uint16(v) | uint16(p.accumulator)<<8
		w >>= bitLen
		v = uint8(w)
		v >>= usedBits - bitLen
	}
	return v, nil
}

func newBitReader(order BitBuilderOrder) bitReader {
	switch order {
	case LSB:
		return bitReader{accInit: initLSB}
	case MSB:
		return bitReader{accInit: initMSB}
	default:
		panic("illegal value")
	}
}

func NewBitIoReader(order BitBuilderOrder, byteReader io.ByteReader) *BitIoReader {
	if byteReader == nil {
		panic("illegal value")
	}
	return &BitIoReader{byteReader: byteReader, bitReader: newBitReader(order)}
}

type BitIoReader struct {
	byteReader io.ByteReader
	bitReader
}

func (p *BitIoReader) ReadBool() (bool, error) {
	v, e := p.ReadNext()
	return v != 0, e
}

func (p *BitIoReader) ReadBit() (int, error) {
	v, e := p.ReadNext()
	if v != 0 {
		return 1, e
	}
	return 0, e
}

func (p *BitIoReader) ReadNext() (int, error) {
	return p.readNext(p.byteReader.ReadByte)
}

func (p *BitIoReader) ReadByte() (byte, error) {
	return p.readByte(p.byteReader.ReadByte)
}

func (p *BitIoReader) ReadSubByte(bitLen uint8) (byte, error) {
	return p.readSubByte(bitLen, p.byteReader.ReadByte)
}

func NewBitArrayReader(direction BitBuilderOrder, bytes []byte) *BitArrayReader {
	return &BitArrayReader{bytes: bytes, bitReader: newBitReader(direction)}
}

type BitArrayReader struct {
	bytes []byte
	bitReader
}

func (p *BitArrayReader) _read() (uint8, error) {
	v := p.bytes[0]
	p.bytes = p.bytes[1:]
	return v, nil
}

func (p *BitArrayReader) IsArrayDepleted() bool {
	return len(p.bytes) == 0
}

func (p *BitArrayReader) ReadBool() (bool, error) {
	v, e := p.ReadNext()
	return v != 0, e
}

func (p *BitArrayReader) ReadBit() (int, error) {
	v, e := p.ReadNext()
	if v != 0 {
		return 1, e
	}
	return 0, e
}

func (p *BitArrayReader) ReadNext() (int, error) {
	return p.readNext(p._read)
}

func (p *BitArrayReader) ReadByte() (byte, error) {
	return p.readByte(p._read)
}

func (p *BitArrayReader) ReadSubByte(bitLen uint8) (byte, error) {
	return p.readSubByte(bitLen, p._read)
}

func NewBitStrReader(direction BitBuilderOrder, bytes ByteString) *BitStrReader {
	return &BitStrReader{bytes: string(bytes), bitReader: newBitReader(direction)}
}

type BitStrReader struct {
	bytes string
	bitReader
}

func (p *BitStrReader) _read() (uint8, error) {
	v := p.bytes[0]
	p.bytes = p.bytes[1:]
	return v, nil
}

func (p *BitStrReader) IsArrayDepleted() bool {
	return len(p.bytes) == 0
}

func (p *BitStrReader) ReadBool() (bool, error) {
	v, e := p.ReadNext()
	return v != 0, e
}

func (p *BitStrReader) ReadBit() (int, error) {
	v, e := p.ReadNext()
	if v != 0 {
		return 1, e
	}
	return 0, e
}

func (p *BitStrReader) ReadNext() (int, error) {
	return p.readNext(p._read)
}

func (p *BitStrReader) ReadByte() (byte, error) {
	return p.readByte(p._read)
}

func (p *BitStrReader) ReadSubByte(bitLen uint8) (byte, error) {
	return p.readSubByte(bitLen, p._read)
}
