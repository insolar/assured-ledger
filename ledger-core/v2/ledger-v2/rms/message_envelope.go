// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import "github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"

type MessageHolder interface {
	MsgBody() GoGoMessage
	PrimaryRecord() RecordHolder
}

func NewMessageEnvelope(cryptProvider PlatformCryptographyProvider, message MessageHolder) MessageEnvelope {
	switch {
	case cryptProvider == nil:
		panic(throw.IllegalValue())
	case message == nil:
		panic(throw.IllegalValue())
	}
	return MessageEnvelope{cryptProvider, message, nil}
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
	if pr := p.message.PrimaryRecord(); pr != nil {
		var b []byte
		err := p._marshalUnmarshal(pr, (*RecordEnvelope).preMarshal, (*RecordEnvelope).postMarshal,
			func(me *InternalMessageEnvelope) (err error) {
				otherRecEnvelopes := p.prepareBundleRecords()

				if len(otherRecEnvelopes) > 0 {
					me.BundleRecords = make([]InternalRecordEnvelope, len(otherRecEnvelopes))
					for i := range otherRecEnvelopes {
						if me.BundleRecords[i], err = otherRecEnvelopes[i].preMarshal(); err != nil {
							return err
						}
					}
				}

				b, err = me.Marshal()
				if err != nil {
					return err
				}

				for i := len(otherRecEnvelopes) - 1; i >= 0; i-- {
					if err = otherRecEnvelopes[i].postMarshal(me.BundleRecords[i]); err != nil {
						return err
					}
				}

				return
			})
		return b, err
	}

	return p.message.MsgBody().Marshal()
}

func (p *MessageEnvelope) Unmarshal(b []byte) error {
	if pr := p.message.PrimaryRecord(); pr != nil {
		err := p._marshalUnmarshal(pr, (*RecordEnvelope).preUnmarshal, (*RecordEnvelope).postUnmarshal,
			func(me *InternalMessageEnvelope) (err error) {
				if err := me.Unmarshal(b); err != nil {
					return err
				}

				if len(me.BundleRecords) > 0 {
					// TODO implement BundleRecords
					return throw.NotImplemented()
				}

				return nil
			})
		return err
	}

	return p.message.MsgBody().Unmarshal(b)
}

func (p *MessageEnvelope) _marshalUnmarshal(pr RecordHolder,
	preFn func(*RecordEnvelope) (InternalRecordEnvelope, error),
	postFn func(*RecordEnvelope, InternalRecordEnvelope) error,
	applyFn func(*InternalMessageEnvelope) error,
) error {
	primaryRecord := NewRecordEnvelope(p.cryptProvider, pr)

	ire, err := preFn(&primaryRecord)
	if err != nil {
		return err
	}

	me := InternalMessageEnvelope{}
	me.MsgBody = interceptor{provider: p.message.MsgBody()}
	me.RecBody = ire.Body
	me.RecExtensions = ire.Extensions

	if err = applyFn(&me); err != nil {
		return err
	}

	// THIS IS DUPLICATED SERIALIZATION
	// as we can't intercept serialization of Record inside Message
	// TODO modify gogoproto codegen to allow interception
	if _, err = ire.Head.triggerMarshalTo(); err != nil {
		return err
	}
	ire.Body = me.RecBody
	ire.Extensions = me.RecExtensions

	if err = postFn(&primaryRecord, ire); err != nil {
		return err
	}
	return nil
}
