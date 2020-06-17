// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/insolar/consensus-reports/pkg/metricreplicator"
	"github.com/insolar/consensus-reports/pkg/middleware"
	"github.com/insolar/consensus-reports/pkg/replicator"
	"github.com/insolar/consensus-reports/pkg/report"
)

type WebDavConfig struct {
	Host      string
	Username  string
	Password  string
	Timeout   time.Duration
	Directory string
}

type PrometheusConfig struct {
	Host string
}

type GitConfig struct {
	Head   string
	Commit string
}

type MetricsConfig struct {
	Quantiles  []string
	Prometheus PrometheusConfig
	WebDav     WebDavConfig
	Git        GitConfig
}

var Groups []middleware.GroupConfig

func CollectMetrics(cfg MetricsConfig, groups []middleware.GroupConfig) (reportDir string, err error) {
	dirName, err := ioutil.TempDir("", "metrics-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dirName)
	repl, err := metricreplicator.New(cfg.Prometheus.Host, dirName)
	if err != nil {
		return "", err
	}

	ctx := context.Background()

	files, charts, err := repl.GrabRecords(ctx, cfg.Quantiles, middleware.GroupsToReplicatorPeriods(groups))
	if err != nil {
		return "", err
	}

	indexFilename := replicator.DefaultConfigFilename
	outputCfg := replicator.OutputConfig{
		Charts:    charts,
		Quantiles: cfg.Quantiles,
	}
	if err := repl.MakeConfigFile(ctx, outputCfg, indexFilename); err != nil {
		return "", err
	}

	files = append(files, indexFilename)

	dirName = fmt.Sprintf("%s/%s-%s-%s", cfg.WebDav.Directory, time.Now().Format("20060102-1504"), cfg.Git.Head, cfg.Git.Commit)
	loaderCfg := replicator.LoaderConfig{
		URL:           cfg.WebDav.Host,
		User:          cfg.WebDav.Username,
		Password:      cfg.WebDav.Password,
		RemoteDirName: dirName,

		Timeout: cfg.WebDav.Timeout,
	}
	if err := repl.UploadFiles(ctx, loaderCfg, files); err != nil {
		return "", err
	}

	return dirName, nil
}

func CreateReport(cfg MetricsConfig, dirName string) (reportURL string, err error) {
	reportCfg := report.Config{
		Webdav: middleware.WebDavConfig{
			Host:      cfg.WebDav.Host,
			Username:  cfg.WebDav.Username,
			Password:  cfg.WebDav.Password,
			Timeout:   cfg.WebDav.Timeout,
			Directory: dirName,
		},
		Git: struct {
			Branch string
			Hash   string
		}{
			Branch: cfg.Git.Head,
			Hash:   cfg.Git.Commit,
		},
	}

	client := report.CreateWebdavClient(reportCfg)
	buf := &bytes.Buffer{}

	err = report.MakeReport(client, buf)
	if err != nil {
		return "", err
	}

	err = client.WriteReport(buf.Bytes())
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s/%s", cfg.WebDav.Host, dirName, report.DefaultReportFileName), nil
}

func init() {
	Groups = append(Groups, middleware.GroupConfig{Description: "Network size grows with zero latency"})
}
