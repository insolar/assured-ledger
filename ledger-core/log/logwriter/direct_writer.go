// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package logwriter

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
)

func NewDirectWriter(output *Adapter) *FatalDirectWriter {
	if output == nil {
		panic("illegal value")
	}

	return &FatalDirectWriter{
		output: output,
	}
}

var _ logcommon.LogLevelWriter = &FatalDirectWriter{}
var _ io.WriteCloser = &FatalDirectWriter{}

type FatalDirectWriter struct {
	output *Adapter
}

func (p *FatalDirectWriter) Close() error {
	return p.output.Close()
}

func (p *FatalDirectWriter) Flush() error {
	return p.output.Flush()
}

func (p *FatalDirectWriter) Write(b []byte) (n int, err error) {
	return p.output.Write(b)
}

func (p *FatalDirectWriter) LowLatencyWrite(level logcommon.Level, b []byte) (int, error) {
	return p.LogLevelWrite(level, b)
}

func (p *FatalDirectWriter) IsLowLatencySupported() bool {
	return false
}

func (p *FatalDirectWriter) LogLevelWrite(level logcommon.Level, b []byte) (n int, err error) {
	switch level {
	case logcommon.FatalLevel:
		if !p.output.SetFatal() {
			break
		}
		n, _ = p.output.DirectLevelWrite(level, b)
		_ = p.output.DirectFlushFatal()
		return n, nil

	case logcommon.PanicLevel:
		n, err = p.output.LogLevelWrite(level, b)
		_ = p.output.Flush()
		return n, err
	}
	return p.output.LogLevelWrite(level, b)
}
