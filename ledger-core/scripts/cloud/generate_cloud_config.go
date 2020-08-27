// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/insolar/defaults"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var (
	debugLevel string
	configDir  string
)

func parseInputParams() {
	var rootCmd = &cobra.Command{}

	rootCmd.Flags().StringVar(
		&configDir, "config-dir", "./", "directory with configs")

	rootCmd.Flags().StringVarP(
		&debugLevel, "debuglevel", "d", "Debug", "debug level")

	err := rootCmd.Execute()
	check("Wrong input params:", err)
}

func check(msg string, err error) {
	if err == nil {
		return
	}

	logCfg := configuration.NewLog()
	logCfg.Formatter = "text"
	inslog, _ := inslogger.NewLog(logCfg)
	inslog.WithField("error", err.Error()).Fatal(msg)
}

func main() {
	parseInputParams()
	generateBaseCloudConfig()
}

func generateBaseCloudConfig() {

	logConfig := configuration.Log{
		Level:     debugLevel,
		Adapter:   "zerolog",
		Formatter: "json",
	}

	foundConfigs, err := filepath.Glob(withBaseDir("discoverynodes/*/insolard.yaml"))
	check("Filed to find configs:", err)
	if len(foundConfigs) < 1 {
		check("Filed to find configs:", throw.New("list is empty"))
	}

	conf := configuration.BaseCloudConfig{
		Log:             logConfig,
		NodeConfigPaths: foundConfigs,
	}

	rawData, err := yaml.Marshal(conf)
	check("Can't Marshal insolard config", err)

	fileName := withBaseDir("configs/base_cloud.yaml")

	err = createFileWithDir(fileName, string(rawData))
	check("failed to create base cloud config: "+fileName, err)
}

func createFileWithDir(file string, text string) error {
	mustMakeDir(filepath.Dir(file))
	return makeFile(file, text)
}

func makeFile(name string, text string) error {
	fmt.Println("generate_cloud_configs.go: write to file", name)
	return ioutil.WriteFile(name, []byte(text), 0644)
}

func mustMakeDir(dir string) {
	err := os.MkdirAll(dir, 0775)
	check("couldn't create directory "+dir, err)
	fmt.Println("generate_cloud_configs.go: creates dir", dir)
}

func baseDir() string {
	return defaults.LaunchnetDir()
}

func withBaseDir(subpath string) string {
	return filepath.Join(baseDir(), subpath)
}
