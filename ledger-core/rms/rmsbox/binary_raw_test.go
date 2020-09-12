// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rmsbox

import (
	"bytes"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

func TestRawBinaryZero(t *testing.T) {
	v := RawBinary{}
	require.True(t, v.IsZero())
	require.True(t, v.IsEmpty())
}

func TestRawBinaryBytes(t *testing.T) {
	suite.Run(t, S(NewRawBytes(nil), nil))
	suite.Run(t, S(NewRawBytes([]byte("abc")), []byte("abc")))
}

func TestRawBinaryByteString(t *testing.T) {
	suite.Run(t, S(NewRaw(longbits.EmptyByteString), nil))
	suite.Run(t, S(NewRaw(longbits.WrapStr("abc")), []byte("abc")))
}

func TestRawBinaryNilMessage(t *testing.T) {
	suite.Run(t, S(NewRawMarshal(nil), nil))
}

func TestRawBinaryMessage(t *testing.T) {
	suite.Run(t, S(NewRawMarshal(&types.BytesValue{}), nil))
	suite.Run(t, S(NewRawMarshal(&types.BytesValue{Value: []byte("abc")}), []byte{0xA, 0x3, 'a', 'b', 'c'}))
}

func TestRawBinaryMessageWithSizer(t *testing.T) {
	suite.Run(t, S(NewRawMarshalWithSizer(&types.Any{}), nil))
	suite.Run(t, S(NewRawMarshal(&types.Any{TypeUrl: "abc"}), []byte{0xA, 0x3, 'a', 'b', 'c'}))
	suite.Run(t, S(NewRawMarshal(&types.Any{Value: []byte("abc")}), []byte{0x12, 0x3, 'a', 'b', 'c'}))
}

func TestRawBinaryCrossEqual(t *testing.T) {
	v1 := NewRawMarshal(&types.Any{TypeUrl: "abc"})
	v2 := NewRawMarshal(&types.BytesValue{Value: []byte("abc")})
	v3 := NewRawBytes([]byte{0xA, 0x3, 'a', 'b', 'c'})

	require.True(t, v1.Equal(&v3))
	require.True(t, v3.Equal(&v1))

	require.True(t, v2.Equal(&v3))
	require.True(t, v3.Equal(&v2))

	require.True(t, v1.Equal(&v2))
	require.True(t, v2.Equal(&v2))

	v1a := NewRawMarshal(&types.Any{TypeUrl: "abc"})
	require.True(t, v1a.Equal(&v1))
	require.True(t, v1.Equal(&v1a))

	v1a = NewRawMarshal(&types.Any{TypeUrl: "abd"})
	require.False(t, v1.Equal(&v1a))
	require.False(t, v1a.Equal(&v1))

	require.False(t, v1a.Equal(&v2))
	require.False(t, v1a.Equal(&v3))
}

func S(value RawBinary, check []byte) *SuiteRawBinary {
	return &SuiteRawBinary{value: value, check: check}
}

type SuiteRawBinary struct {
	suite.Suite

	value RawBinary
	check []byte
}

func (st SuiteRawBinary) TestSize() {
	t := st.T()
	require.Equal(t, len(st.check), st.value.FixedByteSize())
	require.Equal(t, len(st.check), st.value.protoSize())
}

func (st SuiteRawBinary) TestByteString() {
	t := st.T()
	s := longbits.CopyBytes(st.check)
	require.Equal(t, s, st.value.AsByteString())
	require.True(t, longbits.Equal(s, &st.value))
}

func (st SuiteRawBinary) TestBytes() {
	t := st.T()
	require.Equal(t, st.check, st.value.GetBytes())

	expected := st.check
	if len(expected) == 0 {
		expected = []byte{}
	}
	b := make([]byte, len(expected))
	require.Equal(t, len(expected), st.value.CopyTo(b))
	require.Equal(t, expected, b)

	buf := bytes.NewBufferString("")
	n, err := st.value.WriteTo(buf)
	require.NoError(t, err)
	require.Equal(t, int64(buf.Len()), n)
	require.Equal(t, expected, buf.Bytes())
}

func (st SuiteRawBinary) TestMarshalTo() {
	t := st.T()

	size := st.value.protoSize()
	b := make([]byte, size+16)
	n, err := st.value.marshalTo(b)
	require.NoError(t, err)
	require.Equal(t, len(st.check), n)

	r := RawBinary{}
	err = r.unmarshal(b[:n])
	require.NoError(t, err)
	require.True(t, st.value.Equal(&r))
	require.True(t, r.Equal(&st.value))
}

func (st SuiteRawBinary) TestMarshalToSized() {
	t := st.T()

	size := st.value.protoSize()
	b := make([]byte, size+16)
	n, err := st.value.marshalToSizedBuffer(b)
	require.NoError(t, err)
	require.Equal(t, len(st.check), n)

	r := RawBinary{}
	err = r.unmarshal(b[len(b)-n:])
	require.NoError(t, err)
	require.True(t, st.value.Equal(&r))
	require.True(t, r.Equal(&st.value))
}
