// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package example

import (
	"crypto/sha256"
	"fmt"
	"io"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsbox"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

type TestDigester struct {
	altName bool
}

func (p TestDigester) GetDigestMethod() cryptkit.DigestMethod {
	if p.altName {
		return "X-sha224"
	}
	return "sha224"
}

func (p TestDigester) GetDigestSize() int {
	return 224 / 8
}

func (p TestDigester) DigestData(reader io.Reader) cryptkit.Digest {
	h := sha256.New224()
	_, _ = io.Copy(h, reader)
	return cryptkit.DigestOfHash(p, h)
}

func (p TestDigester) DigestBytes(bytes []byte) cryptkit.Digest {
	h := sha256.New224()
	_, _ = h.Write(bytes)
	return cryptkit.DigestOfHash(p, h)
}

func (p TestDigester) NewHasher() cryptkit.DigestHasher {
	return cryptkit.DigestHasher{BasicDigester: p, Hash: sha256.New224()}
}

func TestFieldMap(t *testing.T) {
	ex := RecordExample{
		Str:  NewBytes([]byte("dummy")),           // 40
		Ref1: NewReference(gen.UniqueGlobalRef()), // 41
		AsOf: 234,                                 // 42
	}
	ex.InitFieldMap(true)

	_, err := ex.Marshal()
	if err != nil {
		panic(err)
	}

	fmt.Println(ex.FieldMap.Get(40))
	// [194 2 6 132 100 117 109 109 121]
}

func TestPayloads(t *testing.T) {
	ex := RecordExample{
		Str:  NewBytes([]byte("dummy")),           // 40
		Ref1: NewReference(gen.UniqueGlobalRef()), // 41
		AsOf: 234,                                 // 42
	}
	ex.SetDigester(TestDigester{})

	recordPayload := RawBinary{}
	recordPayload.SetBytes([]byte("payload"))

	extensionPayload1 := RawBinary{}
	extensionPayload1.SetBytes([]byte("payload-extension-1"))

	extensionPayload2 := RawBinary{}
	extensionPayload2.SetBytes([]byte("payload-extension-2"))

	{
		rmsbox.UnsetRecordBodyPayloadsForTest(&ex.RecordBody)

		data, err := ex.Marshal()
		if err != nil {
			panic(err)
		}

		fmt.Println(len(data)) // 56
	}
	{
		rmsbox.UnsetRecordBodyPayloadsForTest(&ex.RecordBody)

		ex.SetPayload(recordPayload)

		data, err := ex.Marshal()
		if err != nil {
			panic(err)
		}

		fmt.Println(len(data)) // 89
	}
	{
		rmsbox.UnsetRecordBodyPayloadsForTest(&ex.RecordBody)

		ex.SetPayload(recordPayload)
		ex.AddExtensionPayload(extensionPayload1)

		data, err := ex.Marshal()
		if err != nil {
			panic(err)
		}

		fmt.Println(len(data)) // 117
	}
	{
		rmsbox.UnsetRecordBodyPayloadsForTest(&ex.RecordBody)

		ex.SetPayload(recordPayload)
		ex.AddExtensionPayload(extensionPayload1)
		ex.AddExtensionPayload(extensionPayload2)

		data, err := ex.Marshal()
		if err != nil {
			panic(err)
		}

		fmt.Println(len(data)) // 145
	}

	data, err := ex.Marshal()
	if err != nil {
		panic(err)
	}

	outRecord := RecordExample{}
	if err := outRecord.Unmarshal(data); err != nil {
		panic(err)
	}
	fmt.Println(outRecord.HasPayloadDigest(), outRecord.GetExtensionDigestCount()) // true, 2
}

func TestPayloadsMarshalUnmarshal(t *testing.T) {
	ex := RecordExample{
		//		RecordBody: RecordBody{},
		Str:  NewBytes([]byte("dummy")),           // 40
		Ref1: NewReference(gen.UniqueGlobalRef()), // 41
		AsOf: 234,                                 // 42
	}

	recordPayload := RawBinary{}
	recordPayload.SetBytes([]byte("payload"))
	ex.SetPayload(recordPayload)

	extensionPayload1 := RawBinary{}
	extensionPayload1.SetBytes([]byte("payload-extension-1"))
	ex.AddExtensionPayload(extensionPayload1)

	extensionPayload2 := RawBinary{}
	extensionPayload2.SetBytes([]byte("payload-extension-2"))
	ex.AddExtensionPayload(extensionPayload2)

	payloads := ex.GetRecordPayloads()

	fmt.Println(payloads.Count()) // 3

	out := make([]byte, payloads.ProtoSize())
	if _, err := payloads.MarshalTo(out); err != nil {
		panic(err)
	}

	fmt.Println(len(out))

	payloadsOut := rmsbox.RecordPayloads{}
	for i := 0; i < 3; i++ {
		if unmarshalledLen, err := payloadsOut.TryUnmarshalPayloadFromBytes(out); err != nil {
			panic(err)
		} else {
			out = out[unmarshalledLen:]
		}
	}

	fmt.Println(payloadsOut.Count()) // 3
}

func TestPayloadWithMessage(t *testing.T) {
	ex := RecordExample{
		Str:  NewBytes([]byte("dummy")),           // 40
		Ref1: NewReference(gen.UniqueGlobalRef()), // 41
		AsOf: 234,                                 // 42
	}
	ex.SetDigester(TestDigester{})
	ex.InitFieldMap(true)

	msg := MessageExample{
		RecordExample: ex,

		MsgParam: 1,
		MsgBytes: NewBytes([]byte("123")),
	}

	data, err := msg.Marshal()
	if err != nil {
		panic(err)
	}

	fmt.Println(len(data))            // 76
	fmt.Println(msg.FieldMap.Get(40)) // [194 2 6 132 100 117 109 109 121]

	data, err = rmsbox.MarshalMessageWithPayloads(&msg)
	if err != nil {
		panic(err)
	}

	fmt.Println(len(data)) // 76

	recordPayload := RawBinary{}
	recordPayload.SetBytes([]byte("payload"))
	ex.SetPayload(recordPayload)

	extensionPayload1 := RawBinary{}
	extensionPayload1.SetBytes([]byte("payload-extension-1"))
	ex.AddExtensionPayload(extensionPayload1)

	extensionPayload2 := RawBinary{}
	extensionPayload2.SetBytes([]byte("payload-extension-2"))
	ex.AddExtensionPayload(extensionPayload2)

	msg.RecordExample = ex

	data, err = rmsbox.MarshalMessageWithPayloads(&msg)
	if err != nil {
		panic(err)
	}

	fmt.Println(len(data)) // 223

	msgOut1 := MessageExample{}
	if err := msgOut1.Unmarshal(data); err != nil {
		panic(err)
	}

	fmt.Println(msgOut1.GetExtensionPayloadCount()) // 0
	fmt.Println(msgOut1.MsgParam)                   // 1

	_, msgOut2Raw, err := rmsbox.UnmarshalMessageWithPayloadsFromBytes(data, TestDigester{}, rmsreg.GetRegistry().Get)
	msgOut2 := msgOut2Raw.(*MessageExample)

	fmt.Println(msgOut2.GetExtensionPayloadCount()) // 2
	fmt.Println(msgOut2.MsgParam)                   // 1
}

func TestMessageWithProjection(t *testing.T) {
	ex := RecordExample{
		Str:  NewBytes([]byte("dummy")),           // 40
		Ref1: NewReference(gen.UniqueGlobalRef()), // 41
		AsOf: 234,                                 // 42
	}
	ex.SetDigester(TestDigester{})
	ex.InitFieldMap(true)

	msg := MessageExample{
		RecordExample: ex,

		MsgParam: 1,
		MsgBytes: NewBytes([]byte("123")),
	}

	data, err := msg.AsHead().Marshal()
	if err != nil {
		panic(err)
	}

	fmt.Println(len(data), data) // 10
}

func TestMessageWithLazyReference(t *testing.T) {
	msg1 := MessageExample{
		RecordExample: RecordExample{
			Str:  NewBytes([]byte("dummy1")),          // 40
			Ref1: NewReference(gen.UniqueGlobalRef()), // 41
			AsOf: 234,                                 // 42
		},
		MsgParam: 1,
		MsgBytes: NewBytes([]byte("123")),
	}

	rmsbox.InitReferenceFactory(&msg1, TestDigester{}, reference.NewSelfRefTemplate(pulse.MinTimePulse, 0))
	rmsbox.SetReferenceFactoryCanPull(&msg1, true)

	lazyRef := rmsbox.DefaultLazyReferenceTo(&msg1)

	msg2 := MessageExample{}
	msg2.RecordExample.Ref1.SetLazy(lazyRef)

	data, err := msg2.Marshal()
	if err != nil {
		panic(err)
	}

	msgOut := MessageExample{}
	if err := msgOut.Unmarshal(data); err != nil {
		panic(err)
	}

	fmt.Println(rmsbox.ForceReferenceOf(&msg1, TestDigester{}, reference.NewSelfRefTemplate(pulse.MinTimePulse, 0)))
	fmt.Println(msgOut.Ref1.GetValue().String()) // identical
}

func TestMessageWithFieldWrappers(t *testing.T) {
	msg1 := MessageExample2{
		Field1:  Any{},
		Field2:  AnyLazy{},
		Field3:  AnyRecord{},
		Field4:  AnyRecordLazy{},
		BinData: NewBytes([]byte("dummy")),
		Local:   NewReferenceLocal(gen.UniqueLocalRef()),
		Global:  NewReference(gen.UniqueGlobalRef()),
	}

	{
		msg1.Field1.Set(&RecordExample{}) // маршалируется когда будет вызыван Marshal
	}

	{
		msg1.Field2.Set(&RecordExample{}) // маршалируется когда будет вызыван Marshal
		// или
		err := msg1.Field2.SetAsLazy(&RecordExample{}) // маршалируется здесь и сейчас
		if err != nil {
			panic(err)
		}
	}

	{
		msg1.Field3.Set(&RecordExample{})
		// есть возможности указать payload
	}

	{
		msg1.Field4.Set(&RecordExample{}) // маршалируется когда будет вызыван Marshal
		// или
		err := msg1.Field4.SetAsLazy(&RecordExample{}) // маршалируется здесь и сейчас
		if err != nil {
			panic(err)
		}
	}
}
