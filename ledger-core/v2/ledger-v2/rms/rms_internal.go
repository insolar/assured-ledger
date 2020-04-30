// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import "github.com/gogo/protobuf/proto"

//go:generate protoc -I=. -I=$GOPATH/src --ins_out=./ rms_internal.proto

type interceptingFunc func([]byte) error

var _ GoGoMarshaller = &interceptor{}
var _ proto.Unmarshaler = &interceptor{}

type interceptor struct {
	provider GoGoMarshaller
	captured []byte
}

func (p *interceptor) ProtoSize() int {
	return p.provider.ProtoSize()
}

func (p *interceptor) MarshalTo(b []byte) (int, error) {
	n, err := p.provider.MarshalTo(b)
	if err == nil {
		p.captured = b[:n]
	}
	return n, err
}

func (p *interceptor) Unmarshal(b []byte) error {
	if p.provider != nil {
		if err := p.provider.Unmarshal(b); err != nil {
			return err
		}
	}
	p.captured = b
	return nil
}

func (p *interceptor) triggerMarshalTo() ([]byte, error) {
	b := make([]byte, p.provider.ProtoSize())
	n, err := p.provider.MarshalTo(b)
	if err != nil {
		return nil, err
	}
	p.captured = b[:n]
	return p.captured, nil
}

func newInternalRecordEnvelope(head GoGoMarshaller, bi interceptorBody, ignoreExtensions bool) InternalRecordEnvelope {
	ire := InternalRecordEnvelope{}
	ire.Head.provider = head
	ire.Body = bi
	if !ignoreExtensions {
		ire.Extensions = bi.extensions
	}
	return ire
}
