package iokit

import (
	"io"
)

func NewTeeWriter(main, copy io.Writer) io.Writer {
	return &TeeWriter{main, teeTemplate{copy, 0}}
}

func NewTeeWriterWithSkip(main, copy io.Writer, zeroLeadingBytes int) io.Writer {
	return &TeeWriter{main, teeTemplate{copy, zeroLeadingBytes}}
}

func NewTeeReader(main io.Reader, copy io.Writer) io.Reader {
	return &TeeReader{main, teeTemplate{copy, 0}}
}

func NewTeeReaderWithSkip(main io.Reader, copy io.Writer, skipLeadingBytes int) io.Reader {
	return &TeeReader{main, teeTemplate{copy, skipLeadingBytes}}
}

type teeTemplate struct {
	CopyTo   io.Writer
	CopySkip int
}

func (w *teeTemplate) teeWrite(n int, b []byte) {
	switch {
	case w.CopyTo == nil:
		return
	case w.CopySkip <= 0:
		_, _ = w.CopyTo.Write(b[:n])
	case w.CopySkip >= n:
		w.CopySkip -= n
	default:
		_, _ = w.CopyTo.Write(b[w.CopySkip:n])
		w.CopySkip = 0
	}
}

/***********************************************/

var _ io.Writer = &TeeWriter{}

type TeeWriter struct {
	W io.Writer
	teeTemplate
}

func (w *TeeWriter) Write(b []byte) (n int, err error) {
	n, err = w.W.Write(b)
	w.teeWrite(n, b)
	return n, err
}

/***********************************************/

var _ io.Reader = &TeeReader{}

type TeeReader struct {
	R io.Reader
	teeTemplate
}

func (w *TeeReader) Read(b []byte) (n int, err error) {
	n, err = w.R.Read(b)
	w.teeWrite(n, b)
	return n, err
}
