// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package logcommon

import (
	"context"
	"time"

	"go.opencensus.io/stats"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
)

func NewMetricsHelper(recorder LogMetricsRecorder) *MetricsHelper {
	return &MetricsHelper{recorder}
}

type MetricsHelper struct {
	recorder LogMetricsRecorder
}

type DurationReportFunc func(d time.Duration)

func (p *MetricsHelper) OnNewEvent(ctx context.Context, level Level) {
	if p == nil {
		return
	}
	if ctx == nil {
		ctx = GetLogLevelContext(level)
	}
	stats.Record(ctx, statLogCalls.M(1))
	if p.recorder != nil {
		p.recorder.RecordLogEvent(level)
	}
}

func (p *MetricsHelper) OnFilteredEvent(ctx context.Context, level Level) {
	if p == nil {
		return
	}
	if ctx == nil {
		ctx = GetLogLevelContext(level)
	}
	stats.Record(ctx, statLogWrites.M(1))
	if p.recorder != nil {
		p.recorder.RecordLogWrite(level)
	}
}

func (p *MetricsHelper) OnWriteDuration(d time.Duration) {
	if p == nil {
		return
	}
	stats.Record(context.Background(), statLogWriteDelays.M(int64(d)))
	if p.recorder != nil {
		p.recorder.RecordLogDelay(Disabled, d)
	}
}

func (p *MetricsHelper) GetOnWriteDurationReport() DurationReportFunc {
	if p == nil {
		return nil
	}
	return p.OnWriteDuration
}

func (p *MetricsHelper) OnWriteSkip(skippedCount int) {
	if p == nil {
		return
	}
	stats.Record(context.Background(), statLogSkips.M(int64(skippedCount)))
}

func (p *MetricsHelper) GetMetricsCollector() logfmt.LogObjectMetricCollector {
	//if p == nil {
	//	return nil
	//}
	//
	return nil
}
