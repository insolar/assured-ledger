package iokit

import (
	"io"
	"math"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type RateLimiter interface {
	TakeQuota(max int64) int64
}

func RateLimitReader(r io.Reader, q RateLimiter) *RateLimitedReader {
	switch {
	case r == nil:
		panic(throw.IllegalValue())
	case q == nil:
		panic(throw.IllegalValue())
	}
	return &RateLimitedReader{r, q}
}

func RateLimitWriter(w io.Writer, q RateLimiter) *RateLimitedWriter {
	switch {
	case w == nil:
		panic(throw.IllegalValue())
	case q == nil:
		panic(throw.IllegalValue())
	}
	return &RateLimitedWriter{w, q}
}

var _ io.Reader = RateLimitedReader{}
var _ io.WriterTo = RateLimitedReader{}

type RateLimitedReader struct {
	R io.Reader
	Q RateLimiter
}

func (r RateLimitedReader) WriteTo(w io.Writer) (int64, error) {
	return RateLimitedCopy(w, r.R, r.Q)
}

func (r RateLimitedReader) Read(p []byte) (int, error) {
	return RateLimitedByteCopy(r.R.Read, p, r.Q)
}

func (r RateLimitedReader) Close() error {
	if c, ok := r.R.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

var _ io.Writer = RateLimitedWriter{}
var _ io.ReaderFrom = RateLimitedWriter{}

type RateLimitedWriter struct {
	W io.Writer
	Q RateLimiter
}

func (r RateLimitedWriter) ReadFrom(rd io.Reader) (int64, error) {
	return RateLimitedCopy(r.W, rd, r.Q)
}

func (r RateLimitedWriter) Write(p []byte) (int, error) {
	return RateLimitedByteCopy(r.W.Write, p, r.Q)
}

func (r RateLimitedWriter) Close() error {
	if c, ok := r.W.(io.Closer); ok {
		return c.Close()
	}
	return nil
}


/****************************/

const rateLimitBlockMin = 4096
const rateLimitBlockMax = 32768 // don't set too high to avoid exhaustion of rate limiter

type quotaSizer struct {
	v int64
}

func (p *quotaSizer) estimate() int64 {
	switch {
	case p.v < rateLimitBlockMin:
		return rateLimitBlockMin
	case p.v > rateLimitBlockMax:
		return rateLimitBlockMax
	}
	return p.v
}

func (p *quotaSizer) update(n int64) {
	if p.v < n {
		p.v = n
	} else {
		p.v -= n
	}
}

func RateLimitedCopy(writer io.Writer, reader io.Reader, q RateLimiter) (int64, error) {
	if q == nil {
		return io.Copy(writer, reader)
	}

	// io.LimitedReader is reused for multiple calls to io.Copy
	sizer := quotaSizer{}
	limited := io.LimitedReader{}
	limited.R, sizer.v = traverseLimitReaders(reader, math.MaxInt64)
	innerW, maxW := traverseLimitWriters(writer, math.MaxInt64)

	switch {
	case maxW == 0 || sizer.v == 0:
		return 0, nil
	case maxW < sizer.v:
		sizer.v = maxW
	case sizer.v == math.MaxInt64:
		sizer.v = 0
	}

	for total := int64(0); ; {
		limited.N += q.TakeQuota(sizer.estimate())
		for limited.N > 0 {
			// io.Copy will use io.ReaderFrom when available
			n, err := io.Copy(innerW, &limited)
			total += n
			if err != nil {
				updatedLimitReaders(reader, total)
				updatedLimitWriters(writer, total)
				return total, err
			}
			sizer.update(n)
		}
	}
}

func RateLimitedByteCopy(fn func([]byte) (int, error), p []byte, q RateLimiter) (int, error) {
	if q == nil {
		return fn(p)
	}

	total := 0
	for len(p) > 0 {
		for q := q.TakeQuota(int64(len(p))); q > 0; {
			n, err := fn(p[:q])
			if err != nil {
				return total + n, err
			}
			total += n
			p = p[n:]
			q -= int64(n)
		}
	}
	return total, nil
}

func RateLimitedBySize(n int64, q RateLimiter) error {
	if q == nil {
		return nil
	}
	for ; n > 0; n -= q.TakeQuota(n) {
	}
	return nil
}
