// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/protokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type RecordPayloads struct {
	payloads []RawBinary
}

func (p *RecordPayloads) ProtoSize() int {
	n := uint64(0)
	wt := protokit.WireBytes.Tag(MessageRecordPayloadsField)
	for _, pl := range p.payloads {
		l := pl.protoSize()
		n += wt.FieldSize(uint64(l))
	}
	return int(n)
}

func (p *RecordPayloads) IsEmpty() bool {
	return len(p.payloads) == 0
}

func (p *RecordPayloads) MarshalTo(b []byte) (int, error) {
	// TODO
	panic(throw.NotImplemented())
	// pos := 0
	// wt := protokit.WireBytes.Tag(MessageRecordPayloadsField)
	// for _, pl := range p.payloads {
	// 	n, err := pl.marshalToSizedBuffer(b[pos:])
	// 	if err != nil {
	// 		return 0, err
	// 	}
	// 	wt.WriteTagValueToBytes(
	// 	l := pl.FixedByteSize()
	// 	n += wt.FieldSize(uint64(l))
	// }
	// return pos, nil
}

func (p *RecordPayloads) TryUnmarshalPayloadFromBytes(b []byte) (int, error) {
	wt := protokit.WireBytes.Tag(MessageRecordPayloadsField)

	minSize := wt.TagSize()
	if len(b) < minSize {
		return 0, nil // it is something else - default skipper should handle it
	}
	u, n := protokit.DecodeVarintFromBytes(b)
	if n != minSize {
		return 0, nil // it is something else
	}
	if err := wt.CheckActualTagValue(u); err != nil {
		return 0, nil // it is something else
	}

	startPos := n
	u, n = protokit.DecodeVarintFromBytes(b[startPos:]) // length of Bytes tag
	startPos += n
	if startPos+int(u) < startPos { // overflow
		return 0, nil // seems to be broken - default skipper should handle it
	}
	endPos := startPos + int(u)
	if n == 0 || endPos > len(b) {
		return 0, nil // seems to be broken - default skipper should handle it
	}

	// end of conditional section. Further fails must be errors

	switch {
	case u == 0:
		p.payloads = append(p.payloads, NewRawBytes(nil))
	case u < protokit.MinPolymorphFieldSize:
		return 0, throw.FailHere("incorrect payload content")
	default:
		p.payloads = append(p.payloads, NewRawBytes(b[startPos:endPos]))
	}
	return endPos, nil
}

func (p *RecordPayloads) ApplyPayloadsTo(record BasicRecord) error {
	// TODO
	panic(throw.NotImplemented())
}
