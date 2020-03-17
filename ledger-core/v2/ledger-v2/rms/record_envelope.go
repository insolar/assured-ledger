// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type RecordEntryExtractorFunc func()

type RecordHolder interface {
	Head() GoGoMarshaller
	Body() *RecordBodyHolder
	EntryExtractor() RecordEntryExtractorFunc
	PolymorphHack
}

func NewRecordEnvelope(cryptProvider PlatformCryptographyProvider, record RecordHolder) RecordEnvelope {
	switch {
	case cryptProvider == nil:
		panic(throw.IllegalValue())
	case record == nil:
		panic(throw.IllegalValue())
	}
	return RecordEnvelope{cryptProvider, record}
}

var _ ProtoMessage = &RecordEnvelope{}

type RecordEnvelope struct {
	cryptProvider PlatformCryptographyProvider
	record        RecordHolder
}

func (p *RecordEnvelope) preMarshal() (InternalRecordEnvelope, error) {
	head := p.record.Head()
	p.record.InitPolymorphField(true)

	body := p.record.Body()

	bi, err := body.beforeAssistedMarshal(p.cryptProvider)
	if err != nil {
		return InternalRecordEnvelope{}, err
	}
	return newInternalRecordEnvelope(head, bi, false), nil
}

func (p *RecordEnvelope) postMarshal(re InternalRecordEnvelope) error {
	body := p.record.Body()
	return body.afterAssistedMarshal(re.Head.captured)
}

func (p *RecordEnvelope) Marshal() ([]byte, error) {
	re, err := p.preMarshal()
	if err != nil {
		return nil, err
	}
	var b []byte
	if b, err = re.Marshal(); err != nil {
		return nil, err
	}
	return b, p.postMarshal(re)
}

func (p *RecordEnvelope) preUnmarshal() (InternalRecordEnvelope, error) {
	head := p.record.Head()
	body := p.record.Body()

	bi, err := body.beforeAssistedUnmarshal(p.cryptProvider)
	if err != nil {
		return InternalRecordEnvelope{}, err
	}
	return newInternalRecordEnvelope(head, bi, true), nil
}

func (p *RecordEnvelope) Unmarshal(b []byte) error {
	re, err := p.preUnmarshal()
	if err != nil {
		return err
	}
	if err = re.Unmarshal(b); err != nil {
		return err
	}
	return p.postUnmarshal(re)
}

func (p *RecordEnvelope) postUnmarshal(re InternalRecordEnvelope) error {
	if !p.record.InitPolymorphField(false) {
		return throw.IllegalValue()
	}
	body := p.record.Body()
	return body.afterAssistedUnmarshal(re.Head.captured, re.Body.extensions, re.Extensions, re.Body.bodyMsg.ExtensionHashes)
}
