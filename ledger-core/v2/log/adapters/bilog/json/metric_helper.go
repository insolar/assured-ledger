//
//    Copyright 2019 Insolar Technologies
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.
//

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
