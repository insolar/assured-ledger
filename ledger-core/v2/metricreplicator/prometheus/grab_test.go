package prometheus

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/metricreplicator/replicator"
)

func TestReplicator_GrabQuantileRecords(t *testing.T) {
	repl := Replicator{TmpDir: "/tmp/tests"}
	repl.APIClient = APIMock{QueryRangeMock: func(ctx context.Context, query string, r v1.Range) (value model.Value, warnings v1.Warnings, err error) {
		var result []*model.SampleStream

		if strings.Contains(query, "metric1") {
			result = []*model.SampleStream{
				{
					Metric: map[model.LabelName]model.LabelValue{"Name": "metric1"},
					Values: []model.SamplePair{
						{Timestamp: 1, Value: 10},
						{Timestamp: 6, Value: 20},
					},
				},
			}
			return model.Matrix(result), nil, nil
		}

		if strings.Contains(query, "metric2") {
			result = []*model.SampleStream{
				{
					Metric: map[model.LabelName]model.LabelValue{"Name": "metric2"},
					Values: []model.SamplePair{
						{Timestamp: 10, Value: 1600},
						{Timestamp: 15, Value: 2800},
					},
				},
			}
			return model.Matrix(result), nil, nil
		}

		if strings.Contains(query, "not_matrix") {
			return model.Vector([]*model.Sample{}), []string{"not matrix"}, nil
		}

		if strings.Contains(query, "query_error") {
			return nil, nil, errors.New("fake query error")
		}

		result = []*model.SampleStream{
			{
				Metric: map[model.LabelName]model.LabelValue{},
				Values: []model.SamplePair{},
			},
			{
				Metric: map[model.LabelName]model.LabelValue{},
				Values: []model.SamplePair{},
			},
		}
		return model.Matrix(result), nil, nil
	}}

	ctx := context.Background()
	ranges := []replicator.RangeParams{
		{
			Start:       time.Unix(1, 0),
			End:         time.Unix(1, 0).Add(5),
			NetworkSize: 5,
		},
		{
			Start:       time.Unix(10, 0),
			End:         time.Unix(10, 0).Add(5),
			NetworkSize: 10,
		},
	}
	params := []replicator.QuantileParams{
		{
			Metric:    "metric1",
			FileName:  "%s_by_network_size.json",
			Quantiles: []string{"0.5", "0.8"},
			Ranges:    ranges,
		},
		{
			Metric:    "metric2",
			FileName:  "%s_by_network_size.json",
			Quantiles: []string{"0.5", "0.8"},
			Ranges:    ranges,
		},
	}

	clean, err := repl.ToTmpDir()
	defer clean()
	require.NoError(t, err)

	t.Run("positive", func(t *testing.T) {
		files, err := repl.GrabQuantileRecords(ctx, params)
		require.NoError(t, err)
		require.Equal(t, []string{"metric1_by_network_size.json", "metric2_by_network_size.json"}, files)
	})
	t.Run("not matrix", func(t *testing.T) {
		params := []replicator.QuantileParams{
			{
				Metric:    "not_matrix",
				FileName:  "%s_by_network_size.json",
				Quantiles: []string{"0.5", "0.8"},
				Ranges:    ranges,
			},
		}
		_, err := repl.GrabQuantileRecords(ctx, params)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unknown type of result")
	})
	t.Run("wrong result length", func(t *testing.T) {
		params := []replicator.QuantileParams{
			{
				Metric:    "fake",
				FileName:  "%s_by_network_size.json",
				Quantiles: []string{"0.5", "0.8"},
				Ranges:    ranges,
			},
		}
		_, err := repl.GrabQuantileRecords(ctx, params)
		require.Error(t, err)
		require.Contains(t, err.Error(), "wrong number of records")
	})
	t.Run("query error", func(t *testing.T) {
		params := []replicator.QuantileParams{
			{
				Metric:    "query_error",
				FileName:  "%s_by_network_size.json",
				Quantiles: []string{"0.5", "0.8"},
				Ranges:    ranges,
			},
		}
		_, err := repl.GrabQuantileRecords(ctx, params)
		require.Error(t, err)
		require.Contains(t, err.Error(), "fake query error")
	})
}
