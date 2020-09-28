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

func MessagePayloadHolder(m BasicMessage) (PayloadHolder, error) {
	if ph, ok := m.(PayloadHolder); ok {
		return ph, nil
	}

	ctx := &msgMarshalContext{m: m}
	if err := m.Visit(ctx); err != nil {
		return nil, err
	}

	return ctx.record, nil
}

func ProtoSizeMessageWithPayloads(m BasicMessage) (sz int) {
	switch ph, err := MessagePayloadHolder(m); {
	case err != nil:
		panic(err)
	case ph != nil:
		if payloads := ph.GetRecordPayloads(); !payloads.IsEmpty() {
			sz = payloads.ProtoSize()
		}
	}
	sz += m.(rmsreg.GoGoSerializable).ProtoSize()
	return sz
}

func MarshalMessageWithPayloadsToBytes(m BasicMessage) ([]byte, error) {
	payloads := RecordPayloads{}
	switch ph, err := MessagePayloadHolder(m); {
	case err != nil:
		panic(err)
	case ph != nil:
		payloads = ph.GetRecordPayloads()
	}

	ms := m.(rmsreg.GoGoSerializable)
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

	switch n, err := ms.MarshalTo(b[pSize:]); {
	case err != nil:
		return nil, err
	case n != mSize:
		panic(throw.IllegalState())
	}

	switch _, polySize, err := protokit.DecodePolymorphFromBytes(b[pSize:], false); {
	case err != nil:
		return nil, throw.W(err, "missing message type")
	default:
		// move first field to the beginning
		copy(b[:polySize], b[pSize:pSize + polySize])

		// insert payload(s) right after the first field
		switch n, err := payloads.MarshalTo(b[polySize:pSize + polySize]); {
		case err != nil:
			return nil, err
		case n != pSize:
			panic(throw.IllegalState())
		}
		return b, nil
	}
}

func UnmarshalMessageWithPayloadsFromBytes(b []byte, digester cryptkit.DataDigester, typeFn rmsreg.UnmarshalTypeFunc) (uint64, BasicMessage, error) {
	payloads := RecordPayloads{}
	id, um, err := rmsreg.UnmarshalCustom(b, typeFn, payloads.TryUnmarshalPayloadFromBytes)
	if err != nil {
		return id, nil, err
	}

	switch m, err := UnmarshalMessageApplyPayloads(id, um, digester, payloads); {
	case err != nil:
		return id, nil, err
	default:
		return id, m, nil
	}
}

func UnmarshalMessageApplyPayloads(id uint64, um interface{}, digester cryptkit.DataDigester, payloads RecordPayloads) (BasicMessage, error) {
	m, ok := um.(BasicMessage)
	if !ok {
		return nil, throw.E("expected BasicMessage", struct{ ID uint64 }{id})
	}

	switch ph, err := MessagePayloadHolder(m); {
	case err != nil:
		panic(err)
	case ph != nil:
		err := payloads.ApplyPayloadsTo(ph, digester)
		return m, err
	case payloads.IsEmpty():
		return m, nil
	default:
		return m, throw.E("message doesn't support payloads", struct{ ID uint64 }{id})
	}
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
	record BasicRecord
}

func (p *msgMarshalContext) Message(m BasicMessage, id uint64) error {
	switch {
	case m != p.m:
		panic(throw.Impossible())
	case id == 0:
		panic(throw.IllegalValue())
	}
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
