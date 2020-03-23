// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package iokit

import (
	"io"
)

func NewTeeWriter(main, copy io.Writer) io.Writer {
	return &TeeWriter{main, teeTemplate{copy, 0}}
}

func NewTeeWriterWithWipe(main, copy io.Writer, zeroLeadingBytes int) io.Writer {
	return &TeeWriter{main, teeTemplate{copy, zeroLeadingBytes}}
}

func NewTeeReader(main io.Reader, copy io.Writer) io.Reader {
	return &TeeReader{main, teeTemplate{copy, 0}}
}

func NewTeeReaderWithWipe(main io.Reader, copy io.Writer, zeroLeadingBytes int) io.Reader {
	return &TeeReader{main, teeTemplate{copy, zeroLeadingBytes}}
}

type teeTemplate struct {
	copy     io.Writer
	zeroHead int
}

func (w *teeTemplate) teeWrite(n int, b []byte) {
	switch {
	case w.copy == nil:
		return
	case w.zeroHead <= 0:
		_, _ = w.copy.Write(b[:n])
	case w.zeroHead >= n:
		w.zeroHead -= n
		_, _ = WriteZeros(n, w.copy)
	default:
		_, _ = WriteZeros(w.zeroHead, w.copy)
		_, _ = w.copy.Write(b[w.zeroHead:n])
		w.zeroHead = 0
	}
}

/***********************************************/

var _ io.Writer = &TeeWriter{}

type TeeWriter struct {
	main io.Writer
	teeTemplate
}

func (w *TeeWriter) Write(b []byte) (n int, err error) {
	n, err = w.main.Write(b)
	w.teeWrite(n, b)
	return n, err
}

/***********************************************/

var _ io.Reader = &TeeReader{}

type TeeReader struct {
	main io.Reader
	teeTemplate
}

func (w *TeeReader) Read(b []byte) (n int, err error) {
	n, err = w.main.Read(b)
	w.teeWrite(n, b)
	return n, err
}
