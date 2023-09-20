package metrics

import (
	"context"
	"expvar"
	"fmt"
	"testing"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
)

var (
	metricFloat = stats.Float64(
		"metric_float",
		"",
		stats.UnitMilliseconds,
	)
	metricInt = stats.Int64(
		"metric_int",
		"",
		"ns",
	)
)

func init() {
	err := view.Register(
		&view.View{
			Name:        metricFloat.Name(),
			Description: metricFloat.Description(),
			Measure:     metricFloat,
			Aggregation: view.Distribution(0.001, 0.01, 0.1, 1, 10, 100, 1000, 5000, 10000, 20000),
		},
		&view.View{
			Name:        metricInt.Name(),
			Description: metricInt.Description(),
			Measure:     metricInt,
			Aggregation: view.Distribution(0.001, 0.01, 0.1, 1, 10, 100, 1000, 5000, 10000, 20000),
		},
	)
	if err != nil {
		panic(err)
	}
}

func Benchmark_Opencensus_MetricInt(b *testing.B) {
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		b.Run(fmt.Sprintf("iteration %v", i), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				start := time.Now()
				stats.Record(ctx, metricInt.M(time.Since(start).Nanoseconds()))
			}
		})
	}
}

func Benchmark_Opencensus_MetricFloat(b *testing.B) {
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		b.Run(fmt.Sprintf("iteration %v", i), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				start := time.Now()
				stats.Record(ctx, metricFloat.M(float64(time.Since(start).Nanoseconds()/1e6)))
			}
		})
	}
}

var phaseTime = expvar.NewInt("Phase01Time")

func Benchmark_Expvar_Metrics(b *testing.B) {
	for i := 0; i < 5; i++ {
		b.Run(fmt.Sprintf("iteration %v", i), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				start := time.Now()
				phaseTime.Set(time.Since(start).Nanoseconds())
			}
		})
	}
}
