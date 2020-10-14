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

type RecordPayloads struct {
	sizes    []int
	payloads []RawBinary

	// for CopyRecordPayloads only
	digester cryptkit.DataDigester
	digests  []cryptkit.Digest
}

func (p *RecordPayloads) isRawBytes() bool {
	return false
}

func (p *RecordPayloads) ProtoSize() int {
	if len(p.sizes) != len(p.payloads) {
		p.sizes = make([]int, len(p.payloads))
		for i := range p.payloads {
			p.sizes[i] = p.payloads[i].protoSize()
		}
	}

	rawBytes := p.isRawBytes()
	n := uint64(0)
	wt := protokit.WireBytes.Tag(rmsreg.MessageRecordPayloadsField)
	for _, sz := range p.sizes {
		if !rawBytes {
			sz = protokit.BinaryProtoSize(sz)
		}
		n += wt.FieldSize(uint64(sz))
	}
	return int(n)
}

func (p *RecordPayloads) IsEmpty() bool {
	return len(p.payloads) == 0
}

func (p *RecordPayloads) MarshalTo(b []byte) (int, error) {
	if len(p.sizes) != len(p.payloads) {
		panic(throw.IllegalState())
	}

	rawBytes := p.isRawBytes()
	pos := 0
	wt := protokit.WireBytes.Tag(rmsreg.MessageRecordPayloadsField)
	for i, pl := range p.payloads {
		sz := p.sizes[i]
		if !rawBytes {
			sz = protokit.BinaryProtoSize(sz)
		}
		// TODO check polymorph prefix when isRawBytes

		n, err := wt.WriteTagValueToBytes(b[pos:], uint64(sz))
		if err != nil {
			return 0, err
		}
		pos += n
		if rawBytes {
			n, err = pl.marshalTo(b[pos : pos+sz])
		} else {
			n, err = protokit.BinaryMarshalTo(b[pos:pos+sz], false, pl.marshalTo)
		}
		switch {
		case err != nil:
			return 0, err
		case n != sz:
			return 0, io.ErrShortWrite
		}
		pos += sz
	}
	return pos, nil
}

func (p *RecordPayloads) TryUnmarshalPayloadFromBytes(b []byte) (int, error) {
	wt := protokit.WireBytes.Tag(rmsreg.MessageRecordPayloadsField)

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

	rawBytes := p.isRawBytes()
	switch {
	case u == 0:
		p.payloads = append(p.payloads, NewRawBytes(nil))
	case !rawBytes:
		// we use binary marker
		if u == 1 || b[startPos] != protokit.BinaryMarker {
			return 0, throw.FailHere("incorrect binary payload content")
		}
		startPos++
	case u < protokit.MinPolymorphFieldSize:
		return 0, throw.FailHere("incorrect polymorph payload content")
	default:
		// TODO check polymorph prefix
	}
	p.payloads = append(p.payloads, NewRawBytes(b[startPos:endPos]))
	return endPos, nil
}

func (p *RecordPayloads) ApplyPayloadsTo(record PayloadHolder, digester cryptkit.DataDigester) error {
	switch {
	case len(p.payloads) == 0:
		if record == nil {
			return nil
		}
	case record == nil:
		return throw.FailHere("unexpected payload(s)")
	}
	return record.SetRecordPayloads(*p, digester)
}

func (p *RecordPayloads) Count() int {
	return len(p.payloads)
}

func (p *RecordPayloads) GetPayloadOrExtension(index int) rmsreg.MarshalerTo {
	payload := p.payloads[index]
	if payload.IsZero() {
		panic(throw.IllegalState())
	}
	return rawBinaryMarshal{payload}
}

func (p *RecordPayloads) GetExtensionID(index int) uint32 {
	switch n := len(p.payloads); {
	case index < 0 || index >= n:
		panic(throw.IllegalValue())
	case index == 0:
		return 0
	default:
		// TODO extract ExtensionID
		return 0
	}
}

func (p *RecordPayloads) WrapSkipFunc(skipFn rmsreg.UnknownCallbackFunc) rmsreg.UnknownCallbackFunc {
	if skipFn == nil {
		return p.TryUnmarshalPayloadFromBytes
	}

	return func(b []byte) (int, error) {
		if n, err := skipFn(b); n != 0 || err != nil {
			return n, err
		}
		return p.TryUnmarshalPayloadFromBytes(b)
	}
}
