package iokit

import "io"

var _ io.Writer = WriterFunc(nil)

type WriterFunc func([]byte) (int, error)

func (v WriterFunc) Write(b []byte) (int, error) {
	if v == nil {
		return 0, nil
	}
	return v(b)
}

var _ io.Reader = ReaderFunc(nil)

type ReaderFunc func([]byte) (int, error)

func (v ReaderFunc) Read(b []byte) (int, error) {
	if v == nil {
		return 0, nil
	}
	return v(b)
}

var _ io.WriterTo = WriterToFunc(nil)

type WriterToFunc func(io.Writer) (int64, error)

func (v WriterToFunc) WriteTo(w io.Writer) (int64, error) {
	if v == nil {
		return 0, nil
	}
	return v(w)
}
