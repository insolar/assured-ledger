package main

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/pflag"

	"github.com/insolar/assured-ledger/ledger-core/v2/metricreplicator/prometheus"
	"github.com/insolar/assured-ledger/ledger-core/v2/metricreplicator/replicator"
)

func main() {
	cfgPath := pflag.String("cfg", "", "config for replicator")
	pflag.Parse()

	if *cfgPath == "" {
		log.Fatalln("empty path to cfg file")
	}

	cfg, err := prometheus.NewConfig(*cfgPath)
	if err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("failed to validate config: %v", err)
	}

	repl, err := prometheus.NewReplicator(cfg.PrometheusHost, cfg.Commit, cfg.TmpDir)
	if err != nil {
		log.Fatalf("failed to init replicator: %v", err)
	}

	if err := Run(repl, cfg); err != nil {
		log.Fatalf("failed to replicate metrics: %v", err)
	}

	fmt.Println("Done!")
}

func Run(repl replicator.Replicator, cfg prometheus.Config) error {
	cleanDir, err := repl.ToTmpDir()
	defer cleanDir()
	if err != nil {
		return err
	}

	filesInfo := cfg.FilesParamsFromConfig()
	quantileParams := cfg.QuantileParamsFromConfig(filesInfo)
	// latencyParams := cfg.LatencyParamsFromConfig(filesInfo)

	ctx := context.Background()

	quantileFiles, err := repl.GrabQuantileRecords(ctx, quantileParams)
	if err != nil {
		return err
	}
	// latencyFiles, err := repl.GrabLatencyRecords(ctx, latencyParams)
	// if err != nil {
	// 	return err
	// }

	// files := append(quantileFiles, latencyFiles...)
	files := quantileFiles
	indexFile, err := repl.MakeIndexFile(ctx, files)
	files = append(files, indexFile)

	pushCfg := cfg.ReplicaConfigFromConfig()
	if err := repl.PushFiles(ctx, pushCfg, files); err != nil {
		return err
	}
	return nil
}
