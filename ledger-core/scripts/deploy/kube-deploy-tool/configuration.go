// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"time"
)

type KubeDeployToolConfig struct {
	NetParams    []NetParams
	KubeParams   KubeParams
	MetricParams MetricsConfig
}

type NetParams struct {
	// number of nodes
	NodesCount uint
	// not working now
	Latency time.Duration
	// time waiting in ready state
	WaitInReadyState time.Duration
	// timeout bootstrap finished
	WaitBootstrap time.Duration
	// timeout get "ready" state
	WaitReady time.Duration
}

type KubeParams struct {
	// local/ci to manage kustomizing for local an ci environment
	Env string
	// path to kube dir
	KubeRootPath string
	// relative path to pure manifests
	ManifestsRelPath string
	// relative path to bootstrap configs, by default in manifests/configuration
	BootstrapConfigRelPath string
	// set to true if needed to leave insolar alive after fail, for debug purposes etc
	LeaveInsolarAliveIfFailed bool
	LogCollector              LogCollector
	Prometheus                Prometheus
	// todo add context specify penv-167
	// you can specify the context for kubectl
	// KubeContext string
	// or you can set the path to your kubectl config
	// KubeConfigPath string
}

type Prometheus struct {
	// start prometheus before test and stop after
	Enabled bool
	// path to prom manifests from KubeRootPath
	ManifestsRelPath string
}

type LogCollector struct {
	// collect logs from nodes after each test
	Enabled bool
	// writable path to save logs, will be cleaned before save
	PathToSave string
}
