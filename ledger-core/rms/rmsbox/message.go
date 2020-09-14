// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rmsbox

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/protokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func MarshalMessageWithPayloadsToBytes(m BasicMessage) ([]byte, error) {
	ms := m.(rmsreg.GoGoSerializable)

	ctx := &msgMarshalContext{m: m}
	if err := m.Visit(ctx); err != nil {
		return nil, err
	}

	payloads := RecordPayloads{}
	switch {
	case ctx.id == 0:
		panic(throw.IllegalValue())
	case ctx.record == nil:
		//
	default:
		payloads = ctx.record.GetRecordPayloads()
	}

	polySize := protokit.GetPolymorphFieldSize(ctx.id)

	mSize := ms.ProtoSize()
	pSize := 0
	if !payloads.IsEmpty() {
		pSize = payloads.ProtoSize()
	}

	if pSize == 0 {
		b := make([]byte, mSize)
		n, err := ms.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}

	b := make([]byte, mSize+pSize)

	switch n, err := payloads.MarshalTo(b[polySize : polySize+pSize]); {
	case err != nil:
		return nil, err
	case n != pSize:
		panic(throw.IllegalState())
	}

	copy(b[:polySize], b[pSize:])

	switch n, err := ms.MarshalTo(b[pSize:]); {
	case err != nil:
		return nil, err
	case n != mSize:
		panic(throw.IllegalState())
	}

	switch pt, id, err := protokit.PeekContentTypeAndPolymorphIDFromBytes(b[pSize : pSize+polySize]); {
	case err != nil:
		panic(throw.W(err, "impossible"))
	case pt != protokit.ContentPolymorph:
		panic(throw.Impossible())
	case id != ctx.id:
		panic(throw.Impossible())
	}

	for i := polySize - 1; i >= 0; i-- {
		b[i], b[pSize+i] = b[pSize+i], b[i]
	}

	return b, nil
}

func UnmarshalMessageWithPayloadsFromBytes(b []byte, digester cryptkit.DataDigester) (uint64, BasicMessage, error) {
	payloads := RecordPayloads{}
	id, um, err := rmsreg.UnmarshalCustom(b, rmsreg.GetRegistry().Get, payloads.TryUnmarshalPayloadFromBytes)
	if err != nil {
		return id, nil, err
	}

	if m, ok := um.(BasicMessage); ok {
		ctx := &msgMarshalContext{m: m}
		if err := m.Visit(ctx); err != nil {
			return id, nil, err
		}

		if err := payloads.ApplyPayloadsTo(ctx.record, digester); err != nil {
			return id, nil, err
		}
		return id, m, err
	}
	return id, nil, throw.E("expected BasicMessage", struct{ ID uint64 }{id})
}

func MarshalMessageWithPayloads(m BasicMessage, w io.Writer) error {
	panic(throw.NotImplemented())
	// TODO Implementation of MarshalMessageWithPayloads must first write payloads into the io.Writer
	// and calculate digests without making full in-memory copy of the payloads.
	// Then use the calculated digests to marshal the message and record.
}

func UnmarshalMessageWithPayloads(m BasicMessage, r io.Reader) (BasicMessage, error) {
	panic(throw.NotImplemented())
	// TODO Implementation of UnmarshalMessageWithPayloads must first read payloads from the io.Reader
	// and calculate digests without making full in-memory copy of the payloads.
	// Then use the calculated digests to unmarshal the message and record.
}

type msgMarshalContext struct {
	m      BasicMessage
	id     uint64
	record BasicRecord
}

func (p *msgMarshalContext) Message(m BasicMessage, id uint64) error {
	switch {
	case m != p.m:
		panic(throw.Impossible())
	case id == 0:
		panic(throw.IllegalValue())
	}
	p.id = id
	return nil
}

func (p *msgMarshalContext) MsgRecord(m BasicMessage, fieldNum int, record BasicRecord) error {
	switch {
	case m != p.m:
		panic(throw.Impossible())
	case fieldNum != rmsreg.RecordBodyField:
		panic(throw.IllegalValue())
	case record == nil:
		panic(throw.IllegalValue())
	case p.record != nil:
		panic(throw.Impossible())
	}
	p.record = record
	return nil
}
