package rms_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsbox"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

func TestAnyLazyRecordWithPayloads(t *testing.T) {
	m := &rms.MessageExample{MsgParam: 11, MsgBytes: []byte("abc")}
	m.Str.Set(longbits.WrapStr("xyz"))

	m.SetDigester(TestDigester{})

	payload := rmsbox.NewRaw(longbits.WrapStr("SomeData"))
	m.SetPayload(payload)
	extension := rmsbox.NewRaw(longbits.WrapStr("SomeExtData"))
	m.AddExtensionPayload(extension)

	m.InitFieldMap(true)
	b, err := m.Marshal()
	require.NoError(t, err)
	require.NotEmpty(t, b)

	recordBytes := m.FieldMap.Message
	require.NotEmpty(t, recordBytes)

	lazyRec := rmsbox.AnyRecordLazy{}

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

	recBody = &br.(*rms.RecordExample).RecordBody
	require.True(t, recBody.HasPayload())
	require.Equal(t, 1, recBody.GetExtensionPayloadCount())
	require.True(t, recBody.IsPostUnmarshalCompleted())
}

func TestAnyRecordLazyInMessage(t *testing.T) {
	m := &rms.MessageExample2{MsgParam: 11, MsgBytes: []byte("abc")}
	e := rms.RecordExample{Str: rms.NewBytes([]byte("---"))}

	require.NoError(t, m.AnyRecordLazy.SetAsLazy(&e))
	b, err := m.Marshal()
	require.NoError(t, err)
	mo := &rms.MessageExample2{}
	require.NoError(t, mo.Unmarshal(b))
}
