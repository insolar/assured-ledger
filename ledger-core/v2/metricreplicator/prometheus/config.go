package prometheus

import (
	"io/ioutil"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/go-playground/validator.v9"
	"gopkg.in/yaml.v2"

	"github.com/insolar/assured-ledger/ledger-core/v2/metricreplicator/replicator"
)

type MetricsConfig struct {
	Traffic []string `yaml:"traffic" validate:"min=1,dive,required"`
	Latency []string `yaml:"latency" validate:"min=1,dive,required"`
}

type PeriodConfig struct {
	NetworkSize uint  `yaml:"network_size" validate:"required"`
	StartTime   int64 `yaml:"start_time" validate:"required"`
}

type ByNetworkSizeConfig struct {
	Interval time.Duration  `yaml:"interval" validate:"required"`
	Periods  []PeriodConfig `yaml:"periods" validate:"required,min=1,dive,required"`
}

type RangesConfig struct {
	ByNetworkSize ByNetworkSizeConfig `yaml:"by_network_size" validate:"required"`
}

type WebDavConfig struct {
	URL      string `yaml:"url" validate:"required"`
	User     string `yaml:"user" validate:"required"`
	Password string `yaml:"password" validate:"required"`
}

type Config struct {
	Metrics        MetricsConfig `yaml:"metrics" validate:"required"`
	Quantiles      []string      `yaml:"quantiles" validate:"min=1,dive,required"`
	TmpDir         string        `yaml:"tmp_directory" validate:"required"`
	PrometheusHost string        `yaml:"prometheus_host" validate:"required"`
	Ranges         RangesConfig  `yaml:"ranges" validate:"required"`
	WebDav         WebDavConfig  `yaml:"webdav" validate:"required"`
	Commit         string        `yaml:"commit_hash" validate:"required"`
}

func NewConfig(cfgPath string) (Config, error) {
	rawData, err := ioutil.ReadFile(cfgPath)
	if err != nil {
		return Config{}, errors.Wrap(err, "failed to read cfg file")
	}

	var cfg Config
	if err := yaml.Unmarshal(rawData, &cfg); err != nil {
		return Config{}, errors.Wrap(err, "failed to unmarshal cfg file")
	}

	return cfg, nil
}

func (cfg *Config) Validate() error {
	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return err
	}
	return nil
}

type FileData struct {
	Ranges   []replicator.RangeParams
	Filename string
}

const (
	byNetworkSizeFileName = "%s_by_network_size.json"
)

func (cfg Config) FilesParamsFromConfig() []FileData {
	result := make([]FileData, 0)

	params := make([]replicator.RangeParams, 0)
	for _, r := range cfg.Ranges.ByNetworkSize.Periods {
		params = append(params, replicator.RangeParams{
			Start:       time.Unix(r.StartTime, 0),
			End:         time.Unix(r.StartTime, 0).Add(cfg.Ranges.ByNetworkSize.Interval),
			NetworkSize: r.NetworkSize,
		})
	}
	result = append(result, FileData{
		Ranges:   params,
		Filename: byNetworkSizeFileName,
	})

	return result
}

func (cfg Config) QuantileParamsFromConfig(filesInfo []FileData) []replicator.QuantileParams {
	params := make([]replicator.QuantileParams, 0)
	for _, f := range filesInfo {
		for _, m := range cfg.Metrics.Traffic {
			params = append(params, replicator.QuantileParams{
				Metric:    m,
				Quantiles: cfg.Quantiles,
				Ranges:    f.Ranges,
				FileName:  f.Filename,
			})
		}
	}
	return params
}

func (cfg Config) LatencyParamsFromConfig(filesInfo []FileData) []replicator.LatencyParams {
	params := make([]replicator.LatencyParams, 0)
	for _, f := range filesInfo {
		for _, m := range cfg.Metrics.Latency {
			params = append(params, replicator.LatencyParams{
				Metric:   m,
				Ranges:   f.Ranges,
				FileName: f.Filename,
			})
		}
	}
	return params
}

func (cfg Config) ReplicaConfigFromConfig() replicator.ReplicaConfig {
	return replicator.ReplicaConfig{
		URL:           cfg.WebDav.URL,
		User:          cfg.WebDav.User,
		Password:      cfg.WebDav.Password,
		RemoteDirName: cfg.Commit,
	}
}
