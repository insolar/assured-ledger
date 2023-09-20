package iokit

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ io.ReadWriteCloser = ErrorWrapper{}
var _ io.ReadWriteSeeker = ErrorWrapper{}
var _ LenReader = ErrorWrapper{}

func WrapError(err error) ErrorWrapper {
	if err != nil {
		panic(throw.IllegalValue())
	}
	return ErrorWrapper{err}
}

type ErrorWrapper struct {
	err error
}

func (w ErrorWrapper) Error() error {
	return w.err
}

func (w ErrorWrapper) Len() int {
	return 1
}

func (w ErrorWrapper) Seek(offset int64, whence int) (int64, error) {
	return 0, w.err
}

func (w ErrorWrapper) Read(p []byte) (n int, err error) {
	return 0, w.err
}

func (w ErrorWrapper) Write(p []byte) (n int, err error) {
	return 0, w.err
}

func (w ErrorWrapper) Close() error {
	return w.err
}
