// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

func TestAnyLazyRecordWithPayloads(t *testing.T) {
	m := &MessageExample{MsgParam: 11, MsgBytes: []byte("abc")}
	m.Str.Set(longbits.WrapStr("xyz"))

	m.SetDigester(TestDigester{})

	payload := NewRaw(longbits.WrapStr("SomeData"))
	m.SetPayload(payload)
	extension := NewRaw(longbits.WrapStr("SomeExtData"))
	m.AddExtensionPayload(extension)

	m.InitFieldMap(true)
	b, err := m.Marshal()
	require.NoError(t, err)
	require.NotEmpty(t, b)

	recordBytes := m.FieldMap.Message
	require.NotEmpty(t, recordBytes)

	lazyRec := AnyRecordLazy{}

	err = lazyRec.Unmarshal(recordBytes)
	require.NoError(t, err)

	isLazy, br := lazyRec.TryGet()
	require.True(t, isLazy)
	require.Nil(t, br)

	lazyVal := lazyRec.TryGetLazy()
	require.False(t, lazyVal.IsEmpty())

	// Polymorph == 0 uses default value on serialization
	// so it has to be set explicitly to equal with a deserialized form
	m.RecordExample.Polymorph = uint32(m.RecordExample.GetDefaultPolymorphID())

	br, err = lazyVal.Unmarshal()
	require.NoError(t, err)
	require.NotNil(t, br)
	require.True(t, m.RecordExample.Equal(br))

	recBody := lazyVal.GetRecordBody()

	require.False(t, recBody.HasPayload())
	require.True(t, recBody.HasPayloadDigest())
	require.Zero(t, recBody.GetExtensionPayloadCount())
	require.Equal(t, 1, recBody.GetExtensionDigestCount())
	require.False(t, recBody.IsPostUnmarshalCompleted())

	recBody.SetDigester(TestDigester{})

	err = recBody.PostUnmarshalVerifyAndAdd(payload)
	require.True(t, recBody.HasPayload())
	require.NoError(t, err)

	err = recBody.PostUnmarshalVerifyAndAdd(extension)
	require.NoError(t, err)
	require.Equal(t, 1, recBody.GetExtensionPayloadCount())
	require.True(t, recBody.IsPostUnmarshalCompleted())

	br, err = lazyVal.Unmarshal()
	require.NoError(t, err)
	require.NotNil(t, br)

	require.True(t, m.RecordExample.Equal(br))

	recBody = &br.(*RecordExample).RecordBody
	require.True(t, recBody.HasPayload())
	require.Equal(t, 1, recBody.GetExtensionPayloadCount())
	require.True(t, recBody.IsPostUnmarshalCompleted())
}
