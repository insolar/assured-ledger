package reference

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/args"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type LocalHash = longbits.Bits224

var nearestPo2 = int(args.CeilingPowerOfTwo(uint(len(LocalHash{}))))

func CopyToLocalHash(r longbits.FixedReader) (result LocalHash) {
	if r == nil {
		panic(throw.IllegalValue())
	}
	if n := r.FixedByteSize(); n != len(result) && n != nearestPo2 {
		panic(throw.IllegalValue())
	}
	r.CopyTo(result[:])
	return
}

func BytesToLocalHash(b []byte) (result LocalHash) {
	if n := len(b); n != len(result) && n != nearestPo2 {
		panic(throw.IllegalValue())
	}
	copy(result[:], b)
	return
}

func NewRecordID(pn pulse.Number, hash LocalHash) Local {
	return NewLocal(pn, 0, hash) // scope is not allowed for RecordID
}

func NewLocal(pn pulse.Number, scope SubScope, hash LocalHash) Local {
	if !pn.IsSpecialOrTimePulse() {
		panic(fmt.Sprintf("illegal value: %d", pn))
	}
	return Local{pulseAndScope: NewLocalHeader(pn, scope), hash: hash}
}

type Local struct {
	pulseAndScope LocalHeader
	hash          LocalHash
}

func (v Local) IsZero() bool {
	return v.pulseAndScope == 0
}

func (v Local) IsEmpty() bool {
	return v.pulseAndScope == 0
}

func (v Local) NotEmpty() bool {
	return !v.IsEmpty()
}

func (v Local) GetPulseNumber() pulse.Number {
	return v.pulseAndScope.Pulse()
}

func (v Local) GetHeader() LocalHeader {
	return v.pulseAndScope
}

func (v Local) SubScope() SubScope {
	return v.pulseAndScope.SubScope()
}

func (v Local) WriteTo(w io.Writer) (int64, error) {
	val := make([]byte, LocalBinaryPulseAndScopeSize)
	v.pulseAndScopeToBytes(val)
	n, err := w.Write(val)
	if err != nil {
		return int64(n), err
	}

	var n2 int64
	n2, err = v.hash.WriteTo(w)
	return int64(n) + n2, err
}

func (v Local) AsByteString() longbits.ByteString {
	return longbits.CopyBytes(v.AsBytes())
}

func (v Local) AsBytes() []byte {
	val := make([]byte, LocalBinarySize)
	v.pulseAndScopeToBytes(val)
	_ = v.hash.CopyTo(val[LocalBinaryPulseAndScopeSize:])
	return val
}

func (v Local) pulseAndScopeToBytes(b []byte) {
	binary.BigEndian.PutUint32(b, uint32(v.pulseAndScope))
}

func pulseAndScopeFromBytes(b []byte) LocalHeader {
	return LocalHeader(binary.BigEndian.Uint32(b))
}

func (v Local) asEncoderReader(limit uint8) *byteReader {
	return &byteReader{v: v, s: limit}
}

// deprecated
func (v Local) ProtoSize() int {
	return BinarySizeLocal(v)
}

// Encoder encodes Local to string with chosen encoder.
func (v Local) Encode(enc Encoder) string {
	repr, err := enc.EncodeRecord(v)
	if err != nil {
		return ""
	}
	return repr
}

func (v Local) String() string {
	return v.pulseAndScope.String() + `/` + v.Encode(DefaultEncoder())
}

func (v Local) Compare(other Local) int {
	if v.pulseAndScope < other.pulseAndScope {
		return -1
	} else if v.pulseAndScope > other.pulseAndScope {
		return 1
	}

	return v.hash.Compare(other.hash)
}

func (v Local) Pulse() pulse.Number {
	return v.GetPulseNumber()
}

func (v Local) IdentityHashBytes() []byte {
	rv := make([]byte, len(v.hash))
	copy(rv, v.hash[:])
	return rv
}

func (v Local) IdentityHash() LocalHash {
	return v.hash
}

func (v Local) hashLen() int {
	i := len(v.hash) - 1
	for ; i >= 0 && v.hash[i] == 0; i-- {
	}
	return i + 1
}

func (v Local) GetLocal() Local {
	return v
}

// DebugString prints ID in human readable form.
func (v Local) DebugString() string {
	return fmt.Sprintf("%s [%d | %d | %s]", v.String(), v.Pulse(), v.SubScope(), base64.RawURLEncoding.EncodeToString(v.IdentityHashBytes()))
}

func (v Local) canConvertToSelf() bool {
	return true
}

func (v Local) Equal(o LocalHolder) bool {
	return o != nil && v == o.GetLocal()
}

func (v Local) WithHash(hash LocalHash) Local {
	v.hash = hash
	return v
}

func (v Local) WithPulse(pn pulse.Number) Local {
	v.pulseAndScope = v.pulseAndScope.WithPulse(pn)
	return v
}

func (v Local) WithSubScope(scope SubScope) Local {
	v.pulseAndScope = v.pulseAndScope.WithSubScope(scope)
	return v
}

func (v Local) WithHeader(h LocalHeader) Local {
	v.pulseAndScope = h
	return v
}

func (v *Local) asDecoderWriter() *byteWriter {
	return &byteWriter{v: v}
}

/********************/

var _ io.ByteWriter = &byteWriter{}

type byteWriter struct {
	v *Local
	o uint8
}

func (p *byteWriter) WriteByte(c byte) error {
	switch {
	case p.o < LocalBinaryPulseAndScopeSize:
		shift := (3 - p.o) << 3
		p.v.pulseAndScope = LocalHeader(c)<<shift | p.v.pulseAndScope&^(0xFF<<shift)
	case p.isFull():
		return io.ErrShortBuffer
	default:
		p.v.hash[p.o-LocalBinaryPulseAndScopeSize] = c
	}
	p.o++
	return nil
}

func (p *byteWriter) isFull() bool {
	return int(p.o) >= LocalBinarySize
}

/********************/

type byteReader struct {
	v Local
	o uint8
	s uint8
}

func (p *byteReader) ReadByte() (b byte, err error) {
	switch {
	case p.o < LocalBinaryPulseAndScopeSize:
		b = byte(p.v.pulseAndScope >> ((3 - p.o) << 3))
	case p.o >= p.s:
		return 0, io.EOF
	default:
		b = p.v.hash[p.o-LocalBinaryPulseAndScopeSize]
	}
	p.o++
	return b, nil
}

func LocalFromString(input string) (Local, error) {
	global, err := DefaultDecoder().Decode(input)
	if err != nil {
		return Local{}, err
	}
	return global.GetLocal(), nil
}
