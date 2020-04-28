package prometheus

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/metricreplicator/replicator"
)

func TestConfig_Validate(t *testing.T) {
	t.Run("all fields", func(t *testing.T) {
		cfg := Config{
			Metrics: MetricsConfig{
				Traffic: []string{"metric1"},
				Latency: []string{"metric2"},
			},
			Quantiles:      []string{"0.5", "0.8"},
			TmpDir:         "/tmp",
			PrometheusHost: "localhost",
			Ranges: RangesConfig{
				ByNetworkSize: ByNetworkSizeConfig{
					Interval: 5,
					Periods: []PeriodConfig{
						{NetworkSize: 5, StartTime: 1},
					},
				},
			},
			WebDav: WebDavConfig{
				URL:      "test",
				User:     "user",
				Password: "pwd",
			},
			Commit: "hash",
		}
		err := cfg.Validate()
		require.NoError(t, err)
	})
	t.Run("without periods", func(t *testing.T) {
		cfg := Config{
			Metrics: MetricsConfig{
				Traffic: []string{"metric1"},
				Latency: []string{"metric2"},
			},
			Quantiles:      []string{"0.5", "0.8"},
			TmpDir:         "/tmp",
			PrometheusHost: "localhost",
			Ranges: RangesConfig{
				ByNetworkSize: ByNetworkSizeConfig{
					Interval: 5,
					Periods:  []PeriodConfig{},
				},
			},
			WebDav: WebDavConfig{
				URL:      "test",
				User:     "user",
				Password: "pwd",
			},
			Commit: "hash",
		}
		err := cfg.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "validation for 'Periods' failed")
	})
	t.Run("without by_network_size", func(t *testing.T) {
		cfg := Config{
			Metrics: MetricsConfig{
				Traffic: []string{"metric1"},
				Latency: []string{"metric2"},
			},
			Quantiles:      []string{"0.5", "0.8"},
			TmpDir:         "/tmp",
			PrometheusHost: "localhost",
			Ranges: RangesConfig{
				ByNetworkSize: ByNetworkSizeConfig{},
			},
			WebDav: WebDavConfig{
				URL:      "test",
				User:     "user",
				Password: "pwd",
			},
			Commit: "hash",
		}
		err := cfg.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "validation for 'Periods' failed")
		require.Contains(t, err.Error(), "validation for 'Interval' failed")
	})
	t.Run("without some fields", func(t *testing.T) {
		cfg := Config{
			Metrics: MetricsConfig{
				Traffic: []string{"metric1"},
				Latency: []string{},
			},
			Quantiles:      []string{"0.5", "0.8"},
			TmpDir:         "/tmp",
			PrometheusHost: "",
			Ranges:         RangesConfig{},
			WebDav: WebDavConfig{
				URL:      "test",
				User:     "user",
				Password: "",
			},
			Commit: "hash",
		}
		err := cfg.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "validation for 'Periods' failed")
		require.Contains(t, err.Error(), "validation for 'Interval' failed")
		require.Contains(t, err.Error(), "validation for 'PrometheusHost' failed")
		require.Contains(t, err.Error(), "validation for 'Latency' failed")
		require.Contains(t, err.Error(), "validation for 'Password' failed")
	})
	t.Run("empty fields", func(t *testing.T) {
		cfg := Config{
			Metrics:        MetricsConfig{},
			Quantiles:      []string{},
			TmpDir:         "",
			PrometheusHost: "",
			Ranges:         RangesConfig{},
			WebDav:         WebDavConfig{},
			Commit:         "",
		}
		err := cfg.Validate()
		require.Error(t, err)
	})
}

func TestNewConfig(t *testing.T) {
	t.Run("positive", func(t *testing.T) {
		cfgPath := "../cmd/config.yml"
		cfg, err := NewConfig(cfgPath)
		require.NoError(t, err)
		require.NotEmpty(t, cfg)
	})
	t.Run("negative", func(t *testing.T) {
		cfgPath := "fake_config.yml"
		_, err := NewConfig(cfgPath)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to read cfg file")
	})
}

func TestConfig_ParamsFromConfig(t *testing.T) {
	cfg := Config{
		Metrics: MetricsConfig{
			Traffic: []string{"metric1", "metric2"},
			Latency: []string{"metric3", "metric4"},
		},
		Quantiles:      []string{"0.5", "0.8"},
		TmpDir:         "/tmp",
		PrometheusHost: "localhost",
		Ranges: RangesConfig{
			ByNetworkSize: ByNetworkSizeConfig{
				Interval: 5,
				Periods: []PeriodConfig{
					{NetworkSize: 5, StartTime: 1},
					{NetworkSize: 10, StartTime: 2},
				},
			},
		},
		WebDav: WebDavConfig{
			URL:      "test",
			User:     "user",
			Password: "pwd",
		},
		Commit: "hash",
	}

	expectedRanges := []replicator.RangeParams{
		{
			Start:       time.Unix(1, 0),
			End:         time.Unix(1, 0).Add(5),
			NetworkSize: 5,
		},
		{
			Start:       time.Unix(2, 0),
			End:         time.Unix(2, 0).Add(5),
			NetworkSize: 10,
		},
	}
	expectedData := []FileData{
		{
			Ranges:   expectedRanges,
			Filename: "%s_by_network_size.json",
		},
	}
	expectedQuantile := []replicator.QuantileParams{
		{
			Metric:    "metric1",
			FileName:  "%s_by_network_size.json",
			Quantiles: []string{"0.5", "0.8"},
			Ranges:    expectedRanges,
		},
		{
			Metric:    "metric2",
			FileName:  "%s_by_network_size.json",
			Quantiles: []string{"0.5", "0.8"},
			Ranges:    expectedRanges,
		},
	}
	expectedLatency := []replicator.LatencyParams{
		{
			Metric:   "metric3",
			FileName: "%s_by_network_size.json",
			Ranges:   expectedRanges,
		},
		{
			Metric:   "metric4",
			FileName: "%s_by_network_size.json",
			Ranges:   expectedRanges,
		},
	}

	filesInfo := cfg.FilesParamsFromConfig()
	require.Equal(t, expectedData, filesInfo)

	quantileParams := cfg.QuantileParamsFromConfig(filesInfo)
	require.Equal(t, expectedQuantile, quantileParams)

	latencyParams := cfg.LatencyParamsFromConfig(filesInfo)
	require.Equal(t, expectedLatency, latencyParams)

	expectedReplica := replicator.ReplicaConfig{
		URL:           "test",
		User:          "user",
		Password:      "pwd",
		RemoteDirName: "hash",
	}
	require.Equal(t, expectedReplica, cfg.ReplicaConfigFromConfig())
}
