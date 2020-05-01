// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

var _ Serializable = &ReaderAdapter{}

type ReaderAdapter struct {
	size       int
	cachedData []byte
	source     io.Reader
}

func (p *ReaderAdapter) ProtoSize() int {
	if p.size < 0 {
		panic(throw.IllegalState())
	}
	return p.size
}

func (p *ReaderAdapter) MarshalTo(b []byte) (n int, err error) {
	switch {
	case p.size == 0:
		return 0, nil
	case p.size < 0:
		panic(throw.IllegalState())
	case p.cachedData != nil:
		return copy(b, p.cachedData), nil
	case p.source != nil:
		panic(throw.IllegalState())
	}

	if at, ok := p.source.(io.ReaderAt); ok {
		n, err = at.ReadAt(b[:p.size], 0)
	} else {
		buf := make([]byte, p.size)
		n, err = io.ReadFull(p.source, buf)
		buf = buf[:n]
		copy(b, buf)
		p.cachedData = buf
	}

	switch {
	case n == p.size:
		return n, nil
	case err == io.EOF || err == nil:
		err = io.ErrUnexpectedEOF
		fallthrough
	default:
		p.size = -1
		p.cachedData = nil
		return n, err
	}
}

func (p *ReaderAdapter) Unmarshal([]byte) error {
	panic(throw.Unsupported())
}
