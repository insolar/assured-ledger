package iokit

import (
	"bytes"
	"io"
	"math"
	"strings"
)

func LimitOfReader(r io.Reader) int64 {
	_, n := traverseLimitReaders(r, math.MaxInt64)
	return n
}

func LimitOfWriter(w io.Writer) int64 {
	_, n := traverseLimitWriters(w, math.MaxInt64)
	return n
}

func LimitOfWriterTo(w io.WriterTo) int64 {
	switch ww := w.(type) {
	case interface{ FixedByteSize() int }:
		return int64(ww.FixedByteSize())
	case interface{ ProtoSize() int }:
		return int64(ww.ProtoSize())
	case interface{ Size() int }:
		return int64(ww.Size())
	case io.Reader:
		if n := sizeOfReader(ww); n >= 0 {
			return n
		}
	}
	return math.MaxInt64
}

func sizeOfReader(r io.Reader) int64 {
	switch rr := r.(type) {
	case *bytes.Buffer:
		return int64(rr.Len())
	case *strings.Reader:
		return int64(rr.Len())
	case interface{ PhysicalRemaining() int64 }:
		return rr.PhysicalRemaining()
	//case interface{ Len() int }:
	//	return int64(rr.Len())
	default:
		return -1
	}
}

func traverseLimitReaders(r io.Reader, min int64) (io.Reader, int64) {
	for r != nil {
		if min <= 0 {
			return r, 0
		}

		var n int64
		switch v := r.(type) {
		case *RateLimitedReader:
			_, min = traverseLimitReaders(v.R, min)
			return r, min
		case *TeeReader:
			if v.CopyTo == nil {
				r = v.R
				continue
			}
			_, min = traverseLimitReaders(v.R, min)
			return v, min
		case *io.LimitedReader:
			n = v.N
			r = v.R
		case *LimitedTriggerReader:
			r = &v.hLimitedReader
			continue
		case *LimitedReader:
			n = v.n
			r = &v.r
		default:
			switch n = sizeOfReader(r); {
			case n >= 0 && n < min:
				return r, n
			default:
				return r, min
			}
		}

		if n < min {
			min = n
		}
	}

	return nil, 0
}

func updatedLimitReaders(r io.Reader, delta int64) {
	if delta <= 0 {
		return
	}

	for r != nil {
		switch v := r.(type) {
		case *TeeReader:
			if v.CopyTo == nil {
				r = v.R
				continue
			}
			return
		case *io.LimitedReader:
			v.N -= delta
			r = v.R
		case *LimitedTriggerReader:
			v.n -= delta
			r = &v.hLimitedReader.r
			if v.n <= 0 {
				// this ensures proper sequence of triggering when there are
				// multiple instances of LimitedTriggerReader on the read chain
				updatedLimitReaders(r, delta)
				v.trigger()
				return
			}
		case *LimitedReader:
			v.n -= delta
			r = &v.r
		default:
			return
		}
	}
}

func traverseLimitWriters(w io.Writer, min int64) (io.Writer, int64) {
	for w != nil {
		if min <= 0 {
			return w, 0
		}

		var n int64
		switch v := w.(type) {
		case *RateLimitedWriter:
			_, min = traverseLimitWriters(v.W, min)
			return w, min
		case *TeeWriter:
			if v.CopyTo == nil {
				w = v.W
				continue
			}
			_, min = traverseLimitWriters(v.W, min)
			return v, min
		case *LimitedWriter:
			n = v.n
			w = &v.w
		default:
			return w, min
		}

		if n < min {
			min = n
		}
	}

	return nil, 0
}

func updatedLimitWriters(w io.Writer, delta int64) {
	if delta <= 0 {
		return
	}

	for w != nil {
		switch v := w.(type) {
		case *TeeWriter:
			if v.CopyTo == nil {
				w = v.W
				continue
			}
			return
		case *LimitedWriter:
			v.n -= delta
			w = &v.w
		default:
			return
		}
	}
}
