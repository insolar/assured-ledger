package rmsbox

import (
	"io"
	"reflect"

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

func MarshalMessageWithPayloads(m BasicMessage) ([]byte, error) {
	return marshalMessageWithPayloadsToBytes(m, nil, true)
}

func MarshalMessageWithPayloadsTo(m BasicMessage, b []byte) (int, error) {
	b2, err := marshalMessageWithPayloadsToBytes(m, b, false)
	return len(b2), err
}

func MarshalMessageWithPayloadsToSizedBuffer(m BasicMessage, b []byte) (int, error) {
	payloads := RecordPayloads{}
	switch ph, err := MessagePayloadHolder(m); {
	case err != nil:
		panic(err)
	case ph != nil:
		payloads = ph.GetRecordPayloads()
	}

	pSize := 0
	if !payloads.IsEmpty() {
		pSize = payloads.ProtoSize()
	}

	ms := m.(rmsreg.GoGoSerializable)

	n, err := ms.MarshalToSizedBuffer(b)
	switch {
	case err != nil:
		return 0, err
	case pSize == 0:
		return n, nil
	}

	nn := n + pSize
	if err := marshalPayloadsToBytes(payloads, b[len(b) - nn:], pSize); err != nil {
		return 0, err
	}
	return nn, nil
}

func marshalMessageWithPayloadsToBytes(m BasicMessage, b []byte, allocate bool) ([]byte, error) {
	payloads := RecordPayloads{}
	switch ph, err := MessagePayloadHolder(m); {
	case err != nil:
		panic(err)
	case ph != nil:
		payloads = ph.GetRecordPayloads()
	}

	pSize := 0
	if !payloads.IsEmpty() {
		pSize = payloads.ProtoSize()
	}

	ms := m.(rmsreg.GoGoSerializable)

	if allocate {
		mSize := ms.ProtoSize()
		b = make([]byte, mSize+pSize)
	}

	n, err := ms.MarshalTo(b[pSize:])
	switch {
	case err != nil:
		return nil, err
	case allocate && n != len(b)-pSize:
		panic(throw.IllegalState())
	case pSize == 0:
		return b, nil
	default:
		b = b[:pSize+n]
	}

	return b, marshalPayloadsToBytes(payloads, b, pSize)
}

func marshalPayloadsToBytes(payloads RecordPayloads, b []byte, offset int) error {
	switch _, headSize, err := protokit.DecodePolymorphFromBytes(b[offset:], false); {
	case err != nil:
		return throw.W(err, "missing message type")
	default:
		// move first field to the beginning
		copy(b[:headSize], b[offset:offset+headSize])

		// insert payload(s) right after the first field
		switch n, err := payloads.MarshalTo(b[headSize:headSize+offset]); {
		case err != nil:
			return err
		case n != offset:
			panic(throw.IllegalState())
		}
		return nil
	}
}

func UnmarshalMessageWithPayloadsFromBytes(b []byte, digester cryptkit.DataDigester, typeFn rmsreg.UnmarshalTypeFunc) (uint64, BasicMessage, error) {
	payloads := RecordPayloads{}
	id, um, err := rmsreg.UnmarshalCustom(b, typeFn, payloads.TryUnmarshalPayloadFromBytes)
	if err != nil {
		return id, nil, err
	}

	switch m, err := UnmarshalMessageApplyPayloads(um, digester, payloads); {
	case err != nil:
		return id, nil, throw.WithDetails(err, struct{ ID uint64 }{id})
	default:
		return id, m, nil
	}
}

func UnmarshalMessageApplyPayloads(um interface{}, digester cryptkit.DataDigester, payloads RecordPayloads) (BasicMessage, error) {
	m, ok := um.(BasicMessage)
	if !ok {
		return nil, throw.E("expected BasicMessage", struct { Type reflect.Type }{ reflect.TypeOf(um) })
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
		return m, throw.E("message doesn't support payloads", struct { Type reflect.Type }{ reflect.TypeOf(um) })
	}
}

func MarshalMessageWithPayloadsToWriter(m BasicMessage, w io.Writer) error {
	panic(throw.NotImplemented())
	// TODO Implementation of MarshalMessageWithPayloads must first write payloads into the io.Writer
	// and calculate digests without making full in-memory copy of the payloads.
	// Then use the calculated digests to marshal the message and record.
}

func UnmarshalMessageWithPayloadsFromReader(m BasicMessage, r io.Reader) (BasicMessage, error) {
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
