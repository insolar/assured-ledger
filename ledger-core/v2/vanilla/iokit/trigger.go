// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package iokit

import "io"

func LimitReaderWithTrigger(r io.Reader, n int64, triggerFn func()) *LimitedTriggerReader {
	return &LimitedTriggerReader{LimitedReader{TeeReader{main: r}, n}, triggerFn}
}

var _ io.ReadCloser = &LimitedTriggerReader{}

type hLimitedReader = LimitedReader
type LimitedTriggerReader struct {
	hLimitedReader
	t func()
}

func (p *LimitedTriggerReader) Read(b []byte) (int, error) {
	if p.hLimitedReader.n <= 0 {
		return 0, io.ErrUnexpectedEOF
	}
	n, err := p.hLimitedReader.Read(b)
	if p.hLimitedReader.n <= 0 {
		//p.hLimitedReader.r.main = nil
		p.t()
	}
	return n, err
}

func (p *LimitedTriggerReader) Close() error {
	panic("implement me")
}
