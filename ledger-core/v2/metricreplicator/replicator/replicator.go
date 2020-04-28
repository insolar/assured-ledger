package replicator

import (
	"context"
	"time"
)

type Replicator interface {
	ToTmpDir() (Clean, error)
	GrabQuantileRecords(ctx context.Context, params []QuantileParams) ([]string, error)
	GrabLatencyRecords(ctx context.Context, params []LatencyParams) ([]string, error)
	MakeIndexFile(ctx context.Context, files []string) (string, error)
	PushFiles(ctx context.Context, cfg ReplicaConfig, files []string) error
}

type RangeParams struct {
	Start       time.Time
	End         time.Time
	NetworkSize uint
	// PacketLossPercentage uint
}

type QuantileParams struct {
	Metric    string
	FileName  string
	Quantiles []string
	Ranges    []RangeParams
}

type LatencyParams struct {
	Metric   string
	FileName string
	Ranges   []RangeParams
}

type ReplicaConfig struct {
	URL           string
	User          string
	Password      string
	RemoteDirName string
}

type Clean func()
