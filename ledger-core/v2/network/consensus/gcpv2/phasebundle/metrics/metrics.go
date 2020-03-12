// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package metrics

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
)

var (
	Phase01Time = stats.Float64(
		"phase01_latency",
		"time spent on phase01",
		stats.UnitMilliseconds,
	)
	Phase02Time = stats.Float64(
		"phase02_latency",
		"time spent on phase02",
		stats.UnitMilliseconds,
	)
	Phase03Time = stats.Float64(
		"phase03_latency",
		"time spent on phase03",
		stats.UnitMilliseconds,
	)
)

func init() {
	err := view.Register(
		&view.View{
			Name:        Phase01Time.Name(),
			Description: Phase01Time.Description(),
			Measure:     Phase01Time,
			Aggregation: view.Distribution(0.001, 0.01, 0.1, 1, 10, 100, 1000, 5000, 10000, 20000),
		},
		&view.View{
			Name:        Phase02Time.Name(),
			Description: Phase02Time.Description(),
			Measure:     Phase02Time,
			Aggregation: view.Distribution(0.001, 0.01, 0.1, 1, 10, 100, 1000, 5000, 10000, 20000),
		},
		&view.View{
			Name:        Phase03Time.Name(),
			Description: Phase03Time.Description(),
			Measure:     Phase03Time,
			Aggregation: view.Distribution(0.001, 0.01, 0.1, 1, 10, 100, 1000, 5000, 10000, 20000),
		},
	)
	if err != nil {
		panic(err)
	}
}
