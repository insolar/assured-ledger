// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

func TestExample(t *testing.T) {
	m := MessageExample{MsgParam: 11}
	require.Equal(t, 999999990, int(m.GetDefaultPolymorphID()))
	require.Equal(t, 999999999, int(m.RecordExample.GetDefaultPolymorphID()))

	require.Equal(t, 11, int(m.GetMsgParam()))
	h := m.AsHeadFace()
	require.Equal(t, 11, int(h.GetMsgParam()))
	require.Equal(t, 999999990, int(h.(*MessageExample_Head).GetDefaultPolymorphID()))
}

func TestExampleUnmarshal(t *testing.T) {
	m := &MessageExample{MsgParam: 11, MsgBytes: []byte("abc")}
	m.Str.Set(longbits.WrapStr("xyz"))
	m.InitFieldMap(true)
	b, err := m.Marshal()
	require.NoError(t, err)
	require.NotEmpty(t, b)

	recordBytes := m.FieldMap.Message
	require.NotEmpty(t, recordBytes)

	// m.SetPayload()

	// Polymorph == 0 uses default value on serialization
	// so it has to be set explicitly to equal with a deserialized form
	m.RecordExample.Polymorph = uint32(m.RecordExample.GetDefaultPolymorphID())

	id, m2, err := Unmarshal(b)
	require.NoError(t, err)
	require.Equal(t, m.GetDefaultPolymorphID(), id)
	require.True(t, m.Equal(m2))

	id, r2, err := Unmarshal(recordBytes)
	require.NoError(t, err)
	require.Equal(t, m.RecordExample.GetDefaultPolymorphID(), id)
	require.True(t, m.RecordExample.Equal(r2))

	head := m.AsHead()

	require.True(t, head == m.AsProjection("Head"))
	require.True(t, m == head.AsProjectionBase())
	require.True(t, m == head.AsMessageExample())

	b, err = head.Marshal()
	require.NoError(t, err)
	require.NotEmpty(t, b)

	id, m2, err = UnmarshalSpecial(b, "Head")
	require.NoError(t, err)
	require.Equal(t, m.GetDefaultPolymorphID(), id)

	require.True(t, m.AsHead().Equal(m2))

	// head doesn't pass all fields
	m.MsgBytes = nil
	m.RecordExample = RecordExample{Str: m.RecordExample.Str}

	require.True(t, m2.(*MessageExample_Head).AsMessageExample().Equal(m))
}

func TestExampleUnmarshalWithSeparatePayload(t *testing.T) {
	m := &MessageExample{MsgParam: 11, MsgBytes: []byte("abc")}
	m.Str.Set(longbits.WrapStr("xyz"))

	m.SetDigester(TestDigester{})

	require.Panics(t, func() { m.SetPayload(RawBinary{}) })
	require.Panics(t, func() { m.AddExtensionPayload(RawBinary{}) })

	require.False(t, m.HasPayload())
	require.False(t, m.HasPayloadDigest())
	payload := NewRaw(longbits.WrapStr("SomeData"))
	m.SetPayload(payload)
	require.True(t, m.HasPayload())
	require.False(t, m.HasPayloadDigest())

	require.Zero(t, m.GetExtensionPayloadCount())
	require.Zero(t, m.GetExtensionDigestCount())
	extension := NewRaw(longbits.WrapStr("SomeExtData"))
	m.AddExtensionPayload(extension)
	require.Equal(t, 1, m.GetExtensionPayloadCount())
	require.Zero(t, m.GetExtensionDigestCount())

	m.InitFieldMap(true)
	b, err := m.Marshal()
	require.NoError(t, err)
	require.NotEmpty(t, b)

	recordBytes := m.FieldMap.Message
	require.NotEmpty(t, recordBytes)

	// Polymorph == 0 uses default value on serialization
	// so it has to be set explicitly to equal with a deserialized form
	m.RecordExample.Polymorph = uint32(m.RecordExample.GetDefaultPolymorphID())

	id, m2, err := Unmarshal(b)
	require.NoError(t, err)
	require.Equal(t, m.GetDefaultPolymorphID(), id)
	require.True(t, m.Equal(m2))

	m2e := m2.(*MessageExample)
	require.False(t, m2e.HasPayload())
	require.True(t, m2e.HasPayloadDigest())
	require.Zero(t, m2e.GetExtensionPayloadCount())
	require.Equal(t, 1, m2e.GetExtensionDigestCount())
	require.False(t, m2e.IsPostUnmarshalCompleted())

	require.True(t, m2e.GetPayload().IsZero())
	require.True(t, m2e.GetExtensionPayload(0).IsZero())

	m2e.SetDigester(TestDigester{})
	err = m2e.PostUnmarshalVerifyAndAdd(payload)
	require.NoError(t, err)

	require.True(t, m2e.HasPayload())
	require.Zero(t, m2e.GetExtensionPayloadCount())
	require.False(t, m2e.GetPayload().IsZero())
	require.True(t, m2e.GetExtensionPayload(0).IsZero())

	err = m2e.PostUnmarshalVerifyAndAdd(extension)
	require.NoError(t, err)
	require.Equal(t, 1, m2e.GetExtensionPayloadCount())
	require.True(t, m2e.IsPostUnmarshalCompleted())

	err = m2e.VerifyAnyPayload(-1, payload)
	require.NoError(t, err)
	err = m2e.VerifyAnyPayload(0, extension)
	require.NoError(t, err)
	err = m2e.VerifyAnyPayload(0, payload)
	require.Error(t, err)
	err = m2e.VerifyAnyPayload(-1, extension)
	require.Error(t, err)

	require.False(t, m2e.GetPayload().IsZero())
	require.False(t, m2e.GetExtensionPayload(0).IsZero())
	require.Panics(t, func() { m2e.GetExtensionPayload(1) })

	id, r2, err := Unmarshal(recordBytes)
	require.NoError(t, err)
	require.Equal(t, m.RecordExample.GetDefaultPolymorphID(), id)
	require.True(t, m.RecordExample.Equal(r2))

	head := m.AsHead()
	b, err = head.Marshal()
	require.NoError(t, err)
	require.NotEmpty(t, b)

	id, m2, err = UnmarshalSpecial(b, "Head")
	require.NoError(t, err)
	require.Equal(t, m.GetDefaultPolymorphID(), id)

	require.True(t, m.AsHead().Equal(m2))

	// head doesn't pass all fields
	m.MsgBytes = nil
	m.RecordExample = RecordExample{Str: m.RecordExample.Str}

	m2e = m2.(*MessageExample_Head).AsMessageExample()
	require.True(t, m2e.Equal(m))

	err = m2e.PostUnmarshalVerifyAndAdd(NewRaw(nil))
	require.NoError(t, err)
	err = m2e.VerifyAnyPayload(-1, NewRaw(nil))
	require.NoError(t, err)

	m2e.SetDigester(TestDigester{})
	err = m2e.PostUnmarshalVerifyAndAdd(NewRaw(nil))
	require.Error(t, err)

	m2e.payloads = nil
	err = m2e.PostUnmarshalVerifyAndAdd(NewRaw(nil))
	require.NoError(t, err)
	err = m2e.VerifyAnyPayload(-1, NewRaw(nil))
	require.NoError(t, err)

	err = m2e.PostUnmarshalVerifyAndAdd(payload)
	require.Error(t, err)
}

func TestExampleRecordRef(t *testing.T) {
	m := &MessageExample{MsgParam: 11, MsgBytes: []byte("abc")}
	m.Str.Set(longbits.WrapStr("xyz"))

	m.SetDigester(TestDigester{})
	payload := NewRaw(longbits.WrapStr("SomeData"))
	m.SetPayload(payload)

	InitReferenceFactory(m, TestDigester{}, reference.NewSelfRefTemplate(pulse.MinTimePulse, 0))

	require.Panics(t, func() { InitReferenceFactory(m, TestDigester{}, reference.NewSelfRefTemplate(pulse.MinTimePulse, 0)) })

	ForceReferenceOf(m, TestDigester{}, reference.NewSelfRefTemplate(pulse.MinTimePulse, 0))
	ForceReferenceOf(m, TestDigester{true}, reference.NewSelfRefTemplate(pulse.MinTimePulse, 0))

	lazyRef := DefaultLazyReferenceTo(m)
	require.True(t, lazyRef.GetReference().IsZero())
	require.True(t, lazyRef.TryPullReference().IsZero())

	lazyRef2 := LazyReferenceTo(m, reference.NewSelfRefTemplate(pulse.MinTimePulse+1, 0))
	require.True(t, lazyRef2.GetReference().IsZero())
	require.True(t, lazyRef2.TryPullReference().IsZero())

	b, err := m.Marshal()
	require.NoError(t, err)
	require.NotEmpty(t, b)

	require.False(t, lazyRef.GetReference().IsZero())
	require.False(t, lazyRef.TryPullReference().IsZero())
	require.Equal(t, pulse.Number(pulse.MinTimePulse), lazyRef.GetReference().GetLocal().Pulse())

	require.False(t, lazyRef2.GetReference().IsZero())
	require.False(t, lazyRef2.TryPullReference().IsZero())
	require.Equal(t, pulse.Number(pulse.MinTimePulse)+1, lazyRef2.GetReference().GetLocal().Pulse())

	require.Equal(t, lazyRef.GetReference().GetLocal().IdentityHash(), lazyRef2.GetReference().GetLocal().IdentityHash())

	ForceReferenceOf(m, TestDigester{}, reference.NewSelfRefTemplate(pulse.MinTimePulse, 0))
	ForceReferenceOf(m, TestDigester{true}, reference.NewSelfRefTemplate(pulse.MinTimePulse, 0))
}

func TestExampleRecordRefPull(t *testing.T) {
	m := &MessageExample{MsgParam: 11, MsgBytes: []byte("abc")}
	m.Str.Set(longbits.WrapStr("xyz"))

	m.SetDigester(TestDigester{})
	payload := NewRaw(longbits.WrapStr("SomeData"))
	m.SetPayload(payload)

	InitReferenceFactory(m, TestDigester{}, reference.NewSelfRefTemplate(pulse.MinTimePulse, 0))
	SetReferenceFactoryCanPull(m, true)

	lazyRef := DefaultLazyReferenceTo(m)
	require.True(t, lazyRef.GetReference().IsZero())
	require.False(t, lazyRef.TryPullReference().IsZero())
	require.False(t, lazyRef.GetReference().IsZero())

	ref := lazyRef.GetReference()
	require.True(t, reference.IsSelfScope(ref))
	require.Equal(t, pulse.MinTimePulse, int(ref.GetLocal().Pulse()))
}

func TestExampleChainedRef(t *testing.T) {
	m := &MessageExample{MsgParam: 11, MsgBytes: []byte("abc")}
	m.Str.Set(longbits.WrapStr("xyz"))
	payload := NewRaw(longbits.WrapStr("SomeData"))
	m.SetPayload(payload)
	m.SetDigester(TestDigester{})

	InitReferenceFactory(m, TestDigester{}, reference.NewSelfRefTemplate(pulse.MinTimePulse, 0))

	m2 := &MessageExample{MsgParam: 11, MsgBytes: []byte("klm")}
	m2.Str.Set(longbits.WrapStr("opq"))
	m2.Ref1.SetLazy(DefaultLazyReferenceTo(m))

	payload2 := NewRaw(longbits.WrapStr("AnotherData"))
	m2.SetPayload(payload2)
	m2.SetDigester(TestDigester{})

	require.Panics(t, func() { _, _ = m2.Marshal() })

	b, err := m.Marshal()
	require.NoError(t, err)
	require.NotEmpty(t, b)

	// Polymorph == 0 uses default value on serialization
	// so it has to be set explicitly to equal with a deserialized form
	m.RecordExample.Polymorph = uint32(m.RecordExample.GetDefaultPolymorphID())

	id, mu, err := Unmarshal(b)
	require.NoError(t, err)
	require.Equal(t, m.GetDefaultPolymorphID(), id)
	require.True(t, m.Equal(mu))

	m2.RecordExample.Polymorph = uint32(m2.RecordExample.GetDefaultPolymorphID())

	b, err = m2.Marshal()
	require.NoError(t, err)
	require.NotEmpty(t, b)
	require.Equal(t, pulse.Number(pulse.MinTimePulse), m2.Ref1.Get().GetLocal().Pulse())

	id, mu, err = Unmarshal(b)
	require.NoError(t, err)
	require.Equal(t, m.GetDefaultPolymorphID(), id)
	require.False(t, m.Equal(mu))
	require.Equal(t, pulse.Number(pulse.MinTimePulse), mu.(*MessageExample).Ref1.Get().GetLocal().Pulse())
	require.True(t, m2.Equal(mu))
}

func TestExampleMessageWithEmbeddedPayloads(t *testing.T) {
	m := &MessageExample{MsgParam: 11, MsgBytes: []byte("abc")}
	m.Str.Set(longbits.WrapStr("xyz"))

	m.SetDigester(TestDigester{})

	payload := NewRaw(longbits.WrapStr("SomeData"))
	m.SetPayload(payload)

	extension := NewRaw(longbits.WrapStr("SomeExtData"))
	m.AddExtensionPayload(extension)

	b, err := MarshalMessageWithPayloadsToBytes(m)

	// Polymorph == 0 uses default value on serialization
	// so it has to be set explicitly to equal with a deserialized form
	m.RecordExample.Polymorph = uint32(m.RecordExample.GetDefaultPolymorphID())

	id, m2, err := UnmarshalMessageWithPayloadsFromBytes(b, TestDigester{})
	require.NoError(t, err)

	require.Equal(t, m.GetDefaultPolymorphID(), id)
	require.True(t, m.Equal(m2))

	m2e := m2.(*MessageExample)
	require.True(t, m2e.HasPayload())
	require.True(t, m2e.HasPayloadDigest())
	require.Equal(t, 1, m2e.GetExtensionDigestCount())
	require.Equal(t, 1, m2e.GetExtensionPayloadCount())
	require.True(t, m2e.IsPostUnmarshalCompleted())

	require.True(t, payload.EqualRaw(m2e.GetPayload()))
	require.True(t, extension.EqualRaw(m2e.GetExtensionPayload(0)))
	require.Panics(t, func() { m2e.GetExtensionPayload(1) })
	require.False(t, extension.EqualRaw(m2e.GetPayload()))
	require.False(t, payload.EqualRaw(m2e.GetExtensionPayload(0)))

	err = m2e.VerifyAnyPayload(-1, payload)
	require.NoError(t, err)
	err = m2e.VerifyAnyPayload(0, extension)
	require.NoError(t, err)
	err = m2e.VerifyAnyPayload(0, payload)
	require.Error(t, err)
	err = m2e.VerifyAnyPayload(-1, extension)
	require.Error(t, err)
}
