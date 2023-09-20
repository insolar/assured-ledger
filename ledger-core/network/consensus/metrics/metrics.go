package metrics

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
)

var (
	Phase01Time = stats.Float64(
		"phase01_latency",
		"time spent on phase 01",
		stats.UnitMilliseconds,
	)
	Phase2Time = stats.Float64(
		"phase2_latency",
		"time spent on phase 2",
		stats.UnitMilliseconds,
	)
	Phase3Time = stats.Float64(
		"phase3_latency",
		"time spent on phase 3",
		stats.UnitMilliseconds,
	)
)

const StatUnit = 1e-6

func init() {
	err := view.Register(
		&view.View{
			Name:        Phase01Time.Name(),
			Description: Phase01Time.Description(),
			Measure:     Phase01Time,
			Aggregation: view.Distribution(0.001, 0.01, 0.1, 1, 10, 100, 1000, 5000, 10000, 20000),
		},
		&view.View{
			Name:        Phase2Time.Name(),
			Description: Phase2Time.Description(),
			Measure:     Phase2Time,
			Aggregation: view.Distribution(0.001, 0.01, 0.1, 1, 10, 100, 1000, 5000, 10000, 20000),
		},
		&view.View{
			Name:        Phase3Time.Name(),
			Description: Phase3Time.Description(),
			Measure:     Phase3Time,
			Aggregation: view.Distribution(0.001, 0.01, 0.1, 1, 10, 100, 1000, 5000, 10000, 20000),
		},
	)
	if err != nil {
		panic(err)
	}
}
