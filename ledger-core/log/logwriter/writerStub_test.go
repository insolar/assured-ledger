package logwriter

import (
	"errors"
	"strings"
)

type TestWriterStub struct {
	strings.Builder
	CloseCount int
	FlushCount int
	NoFlush    bool
}

func (w *TestWriterStub) Close() error {
	w.CloseCount++
	if w.CloseCount > 1 {
		return errClosed
	}
	return nil
}

func (w *TestWriterStub) Flush() error {
	w.FlushCount++
	if w.CloseCount >= 1 {
		return errClosed
	}
	if w.NoFlush {
		return errors.New("unsupported")
	}
	return nil
}
