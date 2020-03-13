// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"io"

	"github.com/gogo/protobuf/proto"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

var _ GoGoMarshaller = &RecordBodyHolder{}

type RecordBodyHolder struct {
	BodyPayload PayloadProvider
	Extensions  []ExtensionProvider

	digester cryptkit.DataDigester
	bodyHash cryptkit.Digest
	headHash *HashDispenser
}

func (m *RecordBodyHolder) ProtoSize() int {
	switch {
	case m.digester == nil:
		panic(throw.IllegalState())
	case m.BodyPayload == nil:
		return 0
	default:
		return m.digester.GetDigestSize()
	}
}

func (m *RecordBodyHolder) MarshalTo(dAtA []byte) (int, error) {
	switch {
	case m.BodyPayload == nil:
		return 0, nil // TODO change to error when optional empty messages will be supported
	case !m.bodyHash.IsEmpty():
		//
	default:
		return 0, throw.IllegalState()
	}
	switch n := m.bodyHash.FixedByteSize(); {
	case n == 0:
		return 0, throw.IllegalState()
	case len(dAtA) < n:
		return 0, io.ErrUnexpectedEOF
	}
	return m.bodyHash.Read(dAtA)
}

func (m *RecordBodyHolder) Unmarshal(b []byte) error {
	if len(b) != m.digester.GetDigestSize() {
		return throw.IllegalValue()
	}
	return m.setHash(cryptkit.NewDigest(longbits.NewMutableFixedSize(b), m.digester.GetDigestMethod()))
}

func (m *RecordBodyHolder) GetHashDispenser() *HashDispenser {
	if m.headHash == nil {
		m.headHash = &HashDispenser{}
	}
	return m.headHash
}

func (m *RecordBodyHolder) unsetHash(head []byte) {
	if hd := m.headHash; hd != nil {
		m.headHash = nil
		hd.set(m.digester.DigestBytes(head))
	}
	m.bodyHash = cryptkit.Digest{}
}

func (m *RecordBodyHolder) setHash(value cryptkit.Digest) error {
	switch {
	case value.IsEmpty():
		return throw.IllegalValue()
	case !m.bodyHash.IsEmpty():
		return throw.IllegalState()
	default:
		m.bodyHash = value
		return nil
	}
}

func (m *RecordBodyHolder) prepareInterceptor(cryptProvider PlatformCryptographyProvider, marshal bool) (interceptorBody, error) {
	body := interceptorBody{bodyHolder: m}

	initExtensions := marshal
	if len(m.Extensions) > 0 {
		body.extensions = make([]interceptor, len(m.Extensions))
		if initExtensions {
			body.bodyMsg.ExtensionHashes = make([]interceptorHash, len(m.Extensions))
		}
	} else {
		initExtensions = false
	}

	bodyScheme := cryptProvider.GetPlatformCryptographyScheme()
	if m.BodyPayload != nil {
		body.bodyMsg.MainContent = m.BodyPayload.GetPayloadContainer(bodyScheme)
	} else {
		body.bodyMsg.MainContent = noMessage{}
	}

	bodySchemeDigester := bodyScheme.GetRecordBodyDigester()
	m.digester = bodySchemeDigester

	for i, ext := range m.Extensions {
		cs := cryptProvider.GetExtensionCryptographyScheme(ext.Extension)
		gp := m.Extensions[i].Provider.GetPayloadContainer(cs)
		body.extensions[i].provider = gp
		if initExtensions {
			body.bodyMsg.ExtensionHashes[i] = interceptorHash{bodySchemeDigester, &body.extensions[i], nil}
		}
	}

	return body, nil
}

func (m *RecordBodyHolder) beforeAssistedMarshal(cryptProvider PlatformCryptographyProvider) (interceptorBody, error) {
	return m.prepareInterceptor(cryptProvider, true)
}

func (m *RecordBodyHolder) afterAssistedMarshal(head []byte) error {
	m.unsetHash(head)
	m.digester = nil
	return nil
}

func (m *RecordBodyHolder) beforeAssistedUnmarshal(cryptProvider PlatformCryptographyProvider) (interceptorBody, error) {
	return m.prepareInterceptor(cryptProvider, false)
}

func (m *RecordBodyHolder) afterAssistedUnmarshal(head []byte, expectedExtensions, actualExtensions []interceptor, actualHashes []interceptorHash) error {
	err := m.checkExtensions(expectedExtensions, actualExtensions, actualHashes)
	m.unsetHash(head)
	m.digester = nil
	return err
}

func (m *RecordBodyHolder) checkExtensions(expectedExtensions, actualExtensions []interceptor, actualHashes []interceptorHash) error {
	switch n, na := len(expectedExtensions), len(actualExtensions); {
	case n == 0:
		return nil
	case n < na:
		return throw.IllegalValue()
	case na != len(actualHashes):
		return throw.IllegalValue()
	default:
		i := 0
		for j, ext := range actualExtensions {
			digest := m.digester.DigestBytes(ext.captured)
			if !longbits.EqualFixedLenWriterToBytes(digest, actualHashes[j].captured) {
				return throw.IllegalValue()
			}

			for {
				err := expectedExtensions[i].provider.Unmarshal(ext.captured)
				i++
				if i >= n {
					return err
				}
			}
		}
	}
	return nil
}

func (m *RecordBodyHolder) afterBodyMarshalled(bi *interceptorBody) error {
	return m.setHash(m.digester.DigestBytes(bi.captured))
}

func (m *RecordBodyHolder) afterBodyUnmarshalled(bi *interceptorBody) error {
	switch {
	case len(bi.captured) == 0:
		if m.bodyHash.IsEmpty() {
			return nil
		}
		return throw.IllegalValue()
	case m.bodyHash.IsEmpty():
		return throw.IllegalState()
	}

	bodyHash := m.digester.DigestBytes(bi.captured)
	if !bodyHash.Equals(m.bodyHash) {
		return throw.IllegalValue()
	}
	return nil
}

var _ GoGoMarshaller = &interceptorBody{}
var _ proto.Unmarshaler = &interceptorBody{}

type interceptorBody struct {
	bodyHolder *RecordBodyHolder

	bodyMsg  InternalRecordBody
	captured []byte

	extensions []interceptor
}

func (p *interceptorBody) ProtoSize() int {
	return p.bodyMsg.ProtoSize()
}

func (p *interceptorBody) MarshalTo(b []byte) (int, error) {
	n, err := p.bodyMsg.MarshalTo(b)
	if err == nil {
		p.captured = b[:n]
		err = p.bodyHolder.afterBodyMarshalled(p)
	}
	return n, err
}

func (p *interceptorBody) Unmarshal(b []byte) error {
	if err := p.bodyMsg.Unmarshal(b); err == nil {
		p.captured = b
		return p.bodyHolder.afterBodyUnmarshalled(p)
	} else {
		return err
	}
}

var _ GoGoMarshaller = &interceptorHash{}
var _ proto.Unmarshaler = &interceptorHash{}

type interceptorHash struct {
	digester cryptkit.DataDigester
	payload  *interceptor
	captured []byte
}

func (p *interceptorHash) ProtoSize() int {
	return p.digester.GetDigestSize()
}

func (p *interceptorHash) MarshalTo(dAtA []byte) (int, error) {
	if len(p.payload.captured) == 0 {
		return 0, throw.IllegalState()
	}
	hash := p.digester.DigestBytes(p.payload.captured)
	if hash.FixedByteSize() > len(dAtA) {
		return 0, io.ErrUnexpectedEOF
	}
	return hash.Read(dAtA)
}

func (p *interceptorHash) Unmarshal(b []byte) error {
	p.captured = b
	return nil
}

var _ GoGoMarshaller = noMessage{}
var _ proto.Unmarshaler = noMessage{}

type noMessage struct {
}

func (p noMessage) ProtoSize() int {
	return 0
}

func (p noMessage) MarshalTo([]byte) (int, error) {
	return 0, nil
}

func (p noMessage) Unmarshal(b []byte) error {
	if len(b) != 0 {
		return throw.IllegalValue()
	}
	return nil
}
