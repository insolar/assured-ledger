// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package json

import (
	"bytes"
	"encoding/binary"
	"io"
	"time"
)

const metricTimeMarker = byte(0)
const metricTimeMarkSize = 9 // SIZE(uint64) + SIZE(metricTimeMarker)

func AppendMetricTimeMark(buf []byte, reportedAt time.Time) []byte {
	if reportedAt.IsZero() {
		return buf
	}
	pos := len(buf)
	if pos > 0 && buf[pos-1] == metricTimeMarker {
		panic("illegal state")
	}

	buf = append(buf, make([]byte, metricTimeMarkSize)...)
	binary.LittleEndian.PutUint64(buf[pos:pos+8], uint64(reportedAt.UnixNano()))
	buf[pos+metricTimeMarkSize-1] = metricTimeMarker
	return buf
}

func CutMetricTimeMarkAndEol(dst *[]byte, eol []byte, now time.Time) (bool, time.Duration) {
	buf := *dst
	pos := len(buf)

	switch {
	case pos < (metricTimeMarkSize + len(eol)):
		return false, 0
	case buf[pos-1] != metricTimeMarker:
		return false, 0
	case len(eol) == 0:
		//
	case !bytes.HasSuffix(buf[:pos-metricTimeMarkSize], eol):
		return false, 0
	}

	timeMark := int64(binary.LittleEndian.Uint64(buf[pos-metricTimeMarkSize:]))
	if now.IsZero() {
		*dst = buf[:pos-metricTimeMarkSize]
		return false, 0
	}

	*dst = buf[:pos-metricTimeMarkSize-len(eol)]
	timeMark = now.UnixNano() - timeMark
	return true, time.Duration(timeMark)
}

var _ io.Writer = MetricTimeWriter{}

type MetricTimeWriter struct {
	Writer   io.Writer
	Eol      []byte
	ReportFn func(time.Duration)
	AppendFn func([]byte, time.Duration) []byte
}

func (w MetricTimeWriter) Write(p []byte) (n int, err error) {
	if ok, d := CutMetricTimeMarkAndEol(&p, w.Eol, time.Now()); ok {
		if w.ReportFn != nil {
			w.ReportFn(d)
		}
		if w.AppendFn != nil {
			p = w.AppendFn(p, d)
		}
		p = append(p, w.Eol...)
	}
	return w.Writer.Write(p)
}
