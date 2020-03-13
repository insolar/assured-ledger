// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reference

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

var byteOrder = binary.BigEndian

type LocalHash = longbits.Bits224

func AsLocalHash(r longbits.FixedReader) (result LocalHash) {
	if r == nil || r.FixedByteSize() != len(result) {
		panic(throw.IllegalValue())
	}
	_, _ = r.Read(result[:])
	return
}

func NewRecordID(pn pulse.Number, hash LocalHash) Local {
	return NewLocal(pn, 0, hash) // scope is not allowed for RecordID
}

func NewLocal(pn pulse.Number, scope SubScope, hash LocalHash) Local {
	if !pn.IsSpecialOrTimePulse() {
		panic(fmt.Sprintf("illegal value: %d", pn))
	}
	return Local{pulseAndScope: pn.WithFlags(byte(scope)), hash: hash}
}

const JetDepthPosition = 0 //

type Local struct {
	pulseAndScope uint32 // pulse + scope
	hash          LocalHash
}

// IsEmpty - check for void
func (v Local) IsEmpty() bool {
	return v.pulseAndScope == 0
}

// NotEmpty - check for non void
func (v Local) NotEmpty() bool {
	return !v.IsEmpty()
}

func (v Local) GetPulseNumber() pulse.Number {
	return pulse.OfUint32(v.pulseAndScope)
}

func (v Local) GetHash() LocalHash {
	return v.hash
}

func (v Local) SubScope() SubScope {
	return SubScope(pulse.FlagsOf(v.pulseAndScope))
}

func (v Local) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(v.pulseAndScopeAsBytes())
	if err != nil {
		return int64(n), err
	}

	var n2 int64
	n2, err = v.hash.WriteTo(w)
	return int64(n) + n2, err
}

func (v Local) Read(p []byte) (n int, err error) {
	if len(p) < LocalBinaryPulseAndScopeSize {
		return copy(p, v.pulseAndScopeAsBytes()), nil
	}

	byteOrder.PutUint32(p, v.pulseAndScope)
	n, err = v.hash.Read(p[LocalBinaryPulseAndScopeSize:])
	if err != nil {
		return 0, err
	}
	return LocalBinaryPulseAndScopeSize + n, nil
}

func (v Local) len() int {
	return LocalBinaryPulseAndScopeSize + len(v.hash)
}

func (v Local) AsByteString() longbits.ByteString {
	return longbits.CopyBytes(v.AsBytes())
}

func (v Local) AsBytes() []byte {
	val := make([]byte, v.len())
	byteOrder.PutUint32(val, v.pulseAndScope)
	_, _ = v.hash.Read(val[LocalBinaryPulseAndScopeSize:])
	return val
}

func (v Local) pulseAndScopeAsBytes() []byte {
	val := make([]byte, LocalBinaryPulseAndScopeSize)
	byteOrder.PutUint32(val, v.pulseAndScope)
	return val
}

func (v Local) AsReader() io.ByteReader {
	return v.asReader(uint8(v.len()))
}

func (v Local) asReader(limit uint8) *byteReader {
	return &byteReader{v: v, s: limit}
}

func (v *Local) asWriter() *byteWriter {
	return &byteWriter{v: v}
}

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

var _ io.ByteWriter = &byteWriter{}

type byteWriter struct {
	v *Local
	o uint8
}

func (p *byteWriter) WriteByte(c byte) error {
	switch {
	case p.o < LocalBinaryPulseAndScopeSize:
		shift := (3 - p.o) << 3
		p.v.pulseAndScope = uint32(c)<<shift | p.v.pulseAndScope&^(0xFF<<shift)
	case p.isFull():
		return io.ErrUnexpectedEOF
	default:
		p.v.hash[p.o-LocalBinaryPulseAndScopeSize] = c
	}
	p.o++
	return nil
}

func (p *byteWriter) isFull() bool {
	return int(p.o) >= p.v.len()
}

// Encoder encodes Local to string with chosen encoder.
func (v Local) Encode(enc Encoder) string {
	repr, err := enc.EncodeRecord(&v)
	if err != nil {
		return ""
	}
	return repr
}

// String implements stringer on ID and returns base64 encoded value
func (v Local) String() string {
	return v.Encode(DefaultEncoder())
}

// Bytes returns byte slice of ID.
func (v Local) Bytes() []byte {
	return v.AsBytes()
}

func (v Local) Compare(other Local) int {
	if v.pulseAndScope < other.pulseAndScope {
		return -1
	} else if v.pulseAndScope > other.pulseAndScope {
		return 1
	}

	return v.hash.Compare(other.hash)
}

// returns a copy of Pulse part of ID.
func (v Local) Pulse() pulse.Number {
	return v.GetPulseNumber()
}

// TODO rename to IdentityHash()
// returns a copy of Hash part of ID
func (v Local) Hash() []byte {
	rv := make([]byte, len(v.hash))
	copy(rv, v.hash[:])
	return rv
}

// MarshalJSON serializes ID into JSONFormat
func (v *Local) MarshalJSON() ([]byte, error) {
	if v == nil {
		return json.Marshal(nil)
	}
	return json.Marshal(v.String())
}

// deprecated
func (v *Local) MarshalBinary() ([]byte, error) {
	return v.Marshal()
}

func (v *Local) Marshal() ([]byte, error) {
	return v.AsBytes(), nil
}

func (v *Local) UnmarshalJSON(data []byte) error {
	var repr interface{}

	err := json.Unmarshal(data, &repr)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal reference.Local")
	}

	switch realRepr := repr.(type) {
	case string:
		decoded, err := DefaultDecoder().Decode(realRepr)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal reference.Local")
		}
		*v = *decoded.GetLocal()
	case nil:
	default:
		return errors.Wrapf(err, "unexpected type %T when unmarshal reference.Local", repr)
	}

	return nil
}

// deprecated
func (v *Local) UnmarshalBinary(data []byte) error {
	return v.Unmarshal(data)
}

func (v Local) ProtoSize() int {
	switch pn := v.GetPulseNumber(); {
	case pn.IsUnknown():
		return 0
	case pn.IsTimePulse():
		return LocalBinarySize
	}
	return LocalBinaryPulseAndScopeSize + v.hashLen()
}

func (v Local) hashLen() int {
	i := len(v.hash) - 1
	for ; i >= 0 && v.hash[i] == 0; i-- {
	}
	return i + 1
}

func (v *Local) MarshalTo(data []byte) (int, error) {
	switch pn := v.GetPulseNumber(); {
	case pn.IsUnknown():
		return 0, nil
	case pn.IsTimePulse():
		if len(data) >= LocalBinarySize {
			return v.Read(data)
		}
	case len(data) >= LocalBinaryPulseAndScopeSize:
		copy(data, v.pulseAndScopeAsBytes())
		i := v.hashLen()
		if copy(data[LocalBinaryPulseAndScopeSize:], v.hash[:i]) == i {
			return LocalBinaryPulseAndScopeSize + i, nil
		}
	}
	return 0, throw.WithStackTop(io.ErrUnexpectedEOF)
}

func (v *Local) Unmarshal(data []byte) error {
	switch n := len(data); {
	case n > LocalBinarySize:
		return errors.New("too much bytes to unmarshal reference.Local")
	case n < LocalBinaryPulseAndScopeSize:
		return errors.New("not enough bytes to unmarshal reference.Local")
	default:
		writer := v.asWriter()
		for i := 0; i < n; i++ {
			_ = writer.WriteByte(data[i])
		}
		for i := n; i < LocalBinarySize; i++ {
			_ = writer.WriteByte(0)
		}
	}
	return nil
}

func (v *Local) FullMarshalTo(data []byte) (int, error) {
	if len(data) < LocalBinarySize {
		return 0, errors.New("not enough bytes to marshal reference.Local")
	}
	return v.Read(data)
}

func (v *Local) FullUnmarshal(b []byte) error {
	if len(b) != LocalBinarySize {
		return errors.New("not enough bytes to marshal reference.Local")
	}

	writer := v.asWriter()
	for i := 0; i < LocalBinarySize; i++ {
		_ = writer.WriteByte(b[i])
	}
	return nil
}

// deprecated
func (v Local) debugStringJet() string {
	depth, prefix := int(v.hash[JetDepthPosition]), v.hash[1:]

	if depth == 0 {
		return "[JET 0 -]"
	} else if depth > 8*(len(v.hash)-1) {
		return fmt.Sprintf("[JET: <wrong format> %d %b]", depth, prefix)
	}

	res := strings.Builder{}
	res.WriteString(fmt.Sprintf("[JET %d ", depth))

	for i := 0; i < depth; i++ {
		bytePos, bitPos := i/8, 7-i%8

		byteValue := prefix[bytePos]
		bitValue := byteValue >> uint(bitPos) & 0x01
		bitString := strconv.Itoa(int(bitValue))
		res.WriteString(bitString)
	}

	res.WriteString("]")
	return res.String()
}

// DebugString prints ID in human readable form.
func (v *Local) DebugString() string {
	if v == nil {
		return NilRef
	} else if v.Pulse().IsJet() {
		// TODO: remove this branch after finish transition to JetID
		return v.debugStringJet()
	}

	return fmt.Sprintf("%s [%d | %d | %s]", v.String(), v.Pulse(), v.SubScope(), base64.RawURLEncoding.EncodeToString(v.Hash()))
}

func (v Local) canConvertToSelf() bool {
	return true
}

func (v Local) Equal(o Local) bool {
	return o == v
}

func (v Local) EqualHolder(o LocalHolder) bool {
	return o != nil && v.EqualPtr(o.GetLocal())
}

func (v Local) EqualPtr(o *Local) bool {
	return o != nil && v.Equal(*o)
}
