package iokit

import "io"

func LimitReaderWithTrigger(r io.Reader, n int64, triggerFn func(int64)) *LimitedTriggerReader {
	return &LimitedTriggerReader{LimitedReader{TeeReader{R: r}, n}, triggerFn}
}

var _ io.ReadCloser = &LimitedTriggerReader{}

type hLimitedReader = LimitedReader
type LimitedTriggerReader struct {
	hLimitedReader
	t func(int64)
}

func (p *LimitedTriggerReader) Read(b []byte) (int, error) {
	if p.hLimitedReader.n <= 0 {
		return 0, io.ErrUnexpectedEOF
	}
	n, err := p.hLimitedReader.Read(b)
	if p.hLimitedReader.n <= 0 {
		p.trigger()
	}
	return n, err
}

func (p *LimitedTriggerReader) trigger() {
	if p.t == nil {
		return
	}
	//p.hLimitedReader.R.W = nil
	t := p.t
	p.t = nil
	t(p.hLimitedReader.n)
}

func (p *LimitedTriggerReader) Close() error {
	p.trigger()
	return nil
}
