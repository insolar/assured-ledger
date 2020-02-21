// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pbuf

import (
	"bytes"
	"encoding/binary"
	"io"
	"time"
)

const metricTimeMarker = byte(0)                         // invalid tag value
const metricTimeMarkMinSize = 1                          // SIZE(metricTimeMarker)
const metricTimeMarkFullSize = metricTimeMarkMinSize + 8 // + SIZE(uint64)

func CutAndUpdateMetricTimeMark(dst *[]byte, now time.Time) (bool, time.Duration) {
	buf := *dst

	tailSize := func() int {
		if len(buf) < int(fieldLogEntry.FieldSize(0))+metricTimeMarkFullSize {
			return 0
		}
		switch entryLen, err := fieldLogEntry.ReadTagValue(bytes.NewBuffer(buf)); {
		case err != nil:
			return 0
		case uint64(len(buf)) <= entryLen:
			return 0
		default:
			switch delta := len(buf) - int(entryLen); delta {
			case metricTimeMarkMinSize, metricTimeMarkFullSize:
				if buf[len(buf)-delta] != metricTimeMarker {
					return 0
				}
				return delta
			default:
				return 0
			}
		}
	}()

	if tailSize == 0 {
		return false, 0
	}

	*dst = buf[:len(buf)-tailSize]
	updateField := false

	switch tailSize {
	case metricTimeMarkMinSize:
		buf = buf[len(buf)-metricTimeMarkFullSize:]
		updateField = true
	case metricTimeMarkFullSize:
		buf = buf[len(buf)-8:]
	default:
		return false, 0
	}

	timeMark := now.UnixNano()
	timeMark -= int64(binary.LittleEndian.Uint64(buf))

	if updateField {
		binary.LittleEndian.PutUint64(buf, uint64(timeMark))
	}

	return true, time.Duration(timeMark)
}

var _ io.Writer = MetricTimeWriter{}

type MetricTimeWriter struct {
	Writer   io.Writer
	ReportFn func(time.Duration)
}

func (w MetricTimeWriter) Write(p []byte) (n int, err error) {
	if ok, d := CutAndUpdateMetricTimeMark(&p, time.Now()); ok {
		if w.ReportFn != nil {
			w.ReportFn(d)
		}
	}
	return w.Writer.Write(p)
}
