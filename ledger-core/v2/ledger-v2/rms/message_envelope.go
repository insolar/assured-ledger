// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"errors"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type MessageHolder interface {
	MsgBody() GoGoMessage
	PrimaryRecord() RecordHolder
	PolymorphHack
}

func NewMessageEnvelope(cryptProvider PlatformCryptographyProvider, message MessageHolder, bundle ...RecordHolder) MessageEnvelope {
	switch {
	case cryptProvider == nil:
		panic(throw.IllegalValue())
	case message == nil:
		panic(throw.IllegalValue())
	}
	return MessageEnvelope{cryptProvider, message, bundle}
}

var _ ProtoMessage = &MessageEnvelope{}

type MessageEnvelope struct {
	cryptProvider PlatformCryptographyProvider
	message       MessageHolder
	bundle        []RecordHolder
}

func (p *MessageEnvelope) prepareBundleRecords() []RecordEnvelope {
	if len(p.bundle) == 0 {
		return nil
	}

	otherRecEnvelopes := make([]RecordEnvelope, len(p.bundle))
	for i, br := range p.bundle {
		otherRecEnvelopes[i] = NewRecordEnvelope(p.cryptProvider, br)
	}
	return otherRecEnvelopes
}

func (p *MessageEnvelope) Marshal() ([]byte, error) {

	p.message.InitPolymorphField(true)

	pr := p.message.PrimaryRecord()
	if pr == nil {
		panic(throw.NotImplemented()) // TODO
		//return p.message.MsgBody().Marshal()
	}

	primaryRecord := NewRecordEnvelope(p.cryptProvider, pr)

	ire, err := primaryRecord.preMarshal()
	if err != nil {
		return nil, err
	}

	var b []byte
	var bundle *interceptorBundle
	var bundleRecs []RecordEnvelope

	if err := p._marshalUnmarshal(pr, &ire,
		func(me *InternalMessageEnvelope) (err error) {
			bundleRecs = p.prepareBundleRecords()

			if len(bundleRecs) > 0 {
				me.BundleRecords = make([]InternalRecordEnvelope, len(bundleRecs))
				for i := range bundleRecs {
					if me.BundleRecords[i], err = bundleRecs[i].preMarshal(); err != nil {
						return err
					}
				}
				bundle = &me.interceptorBundle
			}

			b, err = me.Marshal()
			return
		}); err != nil {
		return nil, err
	}

	if err := primaryRecord.postMarshal(ire); err != nil {
		return nil, err
	}

	if len(bundleRecs) > 0 {
		if err := bundle.postMarshalToSizedBuffer(bundleRecs); err != nil {
			return nil, err
		}
	}

	return b, err
}

func (p *MessageEnvelope) Unmarshal(b []byte) error {
	pr := p.message.PrimaryRecord()
	if pr == nil {
		panic(throw.NotImplemented()) // TODO
		//return p.message.MsgBody().Unmarshal(b)
	}

	primaryRecord := NewRecordEnvelope(p.cryptProvider, pr)

	ire, err := primaryRecord.preUnmarshal()
	if err != nil {
		return err
	}

	if err := p._marshalUnmarshal(pr, &ire,
		func(me *InternalMessageEnvelope) (err error) {
			if err := me.Unmarshal(b); err != nil {
				return err
			}

			if !p.message.InitPolymorphField(false) {
				return throw.IllegalValue()
			}

			if len(me.BundleRecords) > 0 {
				if err := p.unmarshalBundle(me.BundleRecords); err != nil {
					return err
				}
			}

			return nil
		}); err != nil {
		return err
	}

	return primaryRecord.postUnmarshal(ire)
}

func (p *MessageEnvelope) _marshalUnmarshal(pr RecordHolder, ire *InternalRecordEnvelope,
	//postFn func(*RecordEnvelope, InternalRecordEnvelope) error,
	applyFn func(*InternalMessageEnvelope) error,
) error {
	me := InternalMessageEnvelope{}
	me.MsgBody = interceptor{provider: p.message.MsgBody()}
	me.RecBody = ire.Body
	me.RecExtensions = ire.Extensions

	if err := applyFn(&me); err != nil {
		return err
	}

	// THIS IS DUPLICATED SERIALIZATION
	// as we can't intercept serialization of Record inside Message
	// TODO modify gogoproto codegen to allow interception
	if _, err := ire.Head.triggerMarshalTo(); err != nil {
		return err
	}
	ire.Body = me.RecBody
	ire.Extensions = me.RecExtensions

	//if err := postFn(&primaryRecord, ire); err != nil {
	//	return err
	//}
	return nil
}

func (p *MessageEnvelope) unmarshalBundle(records []InternalRecordEnvelope) error {
	p.bundle = make([]RecordHolder, len(records))
	for i, r := range records {
		re := RecordEnvelope{}
		if po, err := PeekPolymorph(r.Head.captured); err != nil {
			return err
		} else if rec, ok := po.(RecordHolder); !ok {
			return throw.Impossible()
		} else {
			re = NewRecordEnvelope(p.cryptProvider, rec)
		}

		if ire, err := re.preUnmarshal(); err != nil {
			return err
		} else {
			ire.Head.captured = r.Head.captured
			if err = ire.Head.Unmarshal(ire.Head.captured); err != nil {
				return err
			}

			ire.Extensions = r.Extensions
			ire.Body.captured = r.Body.captured
			if err = ire.Body.Unmarshal(ire.Body.captured); err != nil {
				return err
			}
			//if err = ire.Body.bodyHolder.afterBodyUnmarshalled(&ire.Body); err != nil {
			//	return err
			//}

			if err = re.postUnmarshal(ire); err != nil {
				return err
			}
		}
		p.bundle[i] = re.record
	}
	return nil
}

var _ GoGoMarshaller = &interceptorBundle{}

type interceptorBundle struct {
	InternalMessageBundle
	sizes    []uint32
	size     int
	captured []byte
}

func (m *interceptorBundle) ProtoSize() int {
	if n := len(m.BundleRecords); n == 0 {
		return 0
	} else {
		m.sizes = make([]uint32, n<<1)
	}
	m.size = 0
	for i, r := range m.BundleRecords {
		sz := r.ProtoSize()
		szLen := sovRmsInternal(uint64(sz))

		m.sizes[i<<1] = uint32(sz)
		m.sizes[1+i<<1] = uint32(szLen)
		m.size += sz + szLen + 2
	}
	return m.size
}

func (m *interceptorBundle) cachedProtoSize() int {
	return m.size
}

func (m *interceptorBundle) MarshalTo(dAtA []byte) (int, error) {
	size := m.cachedProtoSize()
	m.captured = dAtA[:size]
	return size, nil
	//return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *interceptorBundle) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	size := m.cachedProtoSize()
	m.captured = dAtA[len(dAtA)-size:]
	return size, nil
}

func (m *interceptorBundle) postMarshalToSizedBuffer(otherRecEnvelopes []RecordEnvelope) error {
	// this interceptor ensures direct serialization order to allow proper sequence
	// of evaluation of inter-record lazy references

	dAtA := m.captured
	i := 0
	for iNdEx := range m.BundleRecords {
		dAtA[i] = 0xa2
		i++
		dAtA[i] = 0x1
		i++

		j := iNdEx << 1
		size := m.sizes[j]
		i += int(m.sizes[j+1])
		encodeVarintRmsInternal(dAtA, i, uint64(size))

		switch sz, err := m.BundleRecords[iNdEx].MarshalToSizedBuffer(dAtA[i : i+int(size)]); {
		case err != nil:
			return err
		case sz != int(size):
			return errors.New("inconsistent ProtoSize")
		default:
			i += sz
		}
		if err := otherRecEnvelopes[iNdEx].postMarshal(m.BundleRecords[iNdEx]); err != nil {
			return err
		}
	}
	if i != len(dAtA) {
		return io.ErrUnexpectedEOF
	}
	return nil
}
