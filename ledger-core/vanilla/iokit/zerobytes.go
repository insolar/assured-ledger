package iokit

import "io"

var zeroBytes [1024]byte

func WriteZeros(x int, w io.Writer) (int, error) {
	total := 0
	for ; x > len(zeroBytes); x -= len(zeroBytes) {
		n, err := w.Write(zeroBytes[:])
		total += n
		if err != nil {
			return total, err
		}
	}

	n, err := w.Write(zeroBytes[:x])
	total += n
	return total, err
}

var _ io.ReadWriter = NullReadWriter{}
var null = NullReadWriter{}

type NullReadWriter struct{}

func (p NullReadWriter) Read(b []byte) (int, error) {
	for i := range b {
		b[i] = 0
	}
	return len(b), nil
}

func (p NullReadWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func Null() NullReadWriter {
	return null
}
