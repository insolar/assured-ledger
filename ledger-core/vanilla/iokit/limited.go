package iokit

import "io"

type LenReader interface {
	io.Reader
	Len() int
}

func LimitWriter(w io.Writer, n int64) *LimitedWriter {
	return &LimitedWriter{TeeWriter{W: w}, n}
}

func LimitReader(r io.Reader, n int64) *LimitedReader {
	return &LimitedReader{TeeReader{R: r}, n}
}

func NewLimitedTeeWriter(main, copy io.Writer, limit int64) *LimitedWriter {
	return &LimitedWriter{TeeWriter{main, teeTemplate{copy, 0}}, limit}
}

func NewLimitedTeeWriterWithSkip(main, copy io.Writer, skipLeadingBytes int, limit int64) *LimitedWriter {
	return &LimitedWriter{TeeWriter{main, teeTemplate{copy, skipLeadingBytes}}, limit}
}

func NewLimitedTeeReader(main io.Reader, copy io.Writer, limit int64) *LimitedReader {
	return &LimitedReader{TeeReader{main, teeTemplate{copy, 0}}, limit}
}

func NewLimitedTeeReaderWithSkip(main io.Reader, copy io.Writer, skipLeadingBytes int, limit int64) *LimitedReader {
	return &LimitedReader{TeeReader{main, teeTemplate{copy, skipLeadingBytes}}, limit}
}

/***********************************************/

var _ io.Reader = &LimitedReader{}

type LimitedReader struct {
	r TeeReader
	n int64 // max bytes remaining
}

func (l *LimitedReader) Read(p []byte) (n int, err error) {
	if l.n <= 0 {
		return 0, io.ErrUnexpectedEOF
	}
	if int64(len(p)) > l.n {
		p = p[0:l.n]
	}
	n, err = l.r.Read(p)
	l.n -= int64(n)
	return
}

func (l *LimitedReader) RemainingBytes() int64 {
	return l.n
}

func (l *LimitedReader) Len() int {
	return int(l.n)
}

/***********************************************/

var _ io.Writer = &LimitedWriter{}

type LimitedWriter struct {
	w TeeWriter
	n int64 // max bytes remaining
}

func (l *LimitedWriter) Write(p []byte) (n int, err error) {
	if l.n <= 0 {
		return 0, io.ErrShortWrite
	}
	if int64(len(p)) > l.n {
		p = p[0:l.n]
	}
	n, err = l.w.Write(p)
	l.n -= int64(n)
	return
}

func (l *LimitedWriter) RemainingBytes() int64 {
	return l.n
}
