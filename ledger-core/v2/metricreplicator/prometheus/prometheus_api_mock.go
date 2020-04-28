package prometheus

import (
	"context"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type APIMock struct {
	QueryRangeMock func(ctx context.Context, query string, r v1.Range) (model.Value, v1.Warnings, error)
}

func (m APIMock) Alerts(ctx context.Context) (v1.AlertsResult, error) {
	panic("implement me")
}

func (m APIMock) AlertManagers(ctx context.Context) (v1.AlertManagersResult, error) {
	panic("implement me")
}

func (m APIMock) CleanTombstones(ctx context.Context) error {
	panic("implement me")
}

func (m APIMock) Config(ctx context.Context) (v1.ConfigResult, error) {
	panic("implement me")
}

func (m APIMock) DeleteSeries(ctx context.Context, matches []string, startTime time.Time, endTime time.Time) error {
	panic("implement me")
}

func (m APIMock) Flags(ctx context.Context) (v1.FlagsResult, error) {
	panic("implement me")
}

func (m APIMock) LabelNames(ctx context.Context) ([]string, v1.Warnings, error) {
	panic("implement me")
}

func (m APIMock) LabelValues(ctx context.Context, label string) (model.LabelValues, v1.Warnings, error) {
	panic("implement me")
}

func (m APIMock) Query(ctx context.Context, query string, ts time.Time) (model.Value, v1.Warnings, error) {
	panic("implement me")
}

func (m APIMock) QueryRange(ctx context.Context, query string, r v1.Range) (model.Value, v1.Warnings, error) {
	return m.QueryRangeMock(ctx, query, r)
}

func (m APIMock) Series(ctx context.Context, matches []string, startTime time.Time, endTime time.Time) ([]model.LabelSet, v1.Warnings, error) {
	panic("implement me")
}

func (m APIMock) Snapshot(ctx context.Context, skipHead bool) (v1.SnapshotResult, error) {
	panic("implement me")
}

func (m APIMock) Rules(ctx context.Context) (v1.RulesResult, error) {
	panic("implement me")
}

func (m APIMock) Targets(ctx context.Context) (v1.TargetsResult, error) {
	panic("implement me")
}

func (m APIMock) TargetsMetadata(ctx context.Context, matchTarget string, metric string, limit string) ([]v1.MetricMetadata, error) {
	panic("implement me")
}

func (m APIMock) Metadata(ctx context.Context, metric string, limit string) (map[string][]v1.Metadata, error) {
	panic("implement me")
}
