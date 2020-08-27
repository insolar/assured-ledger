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
	numVirtual int
)

func parseInputParams() {
	var rootCmd = &cobra.Command{}

	rootCmd.Flags().IntVar(
		&numVirtual, "num-virtual-nodes", 5, "number of nodes with role virtual")

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
	if numVirtual < 1 {
		check("cannot generate config", throw.New("virtual count must be more than zero"))
	}

	logConfig := configuration.Log{
		Level:     debugLevel,
		Adapter:   "zerolog",
		Formatter: "json",
	}

	nodeConfigPathTmpl := ".artifacts/launchnet/discoverynodes/%d/insolard.yaml"
	nodeConfigs := []string{}
	for i := 1; i <= numVirtual; i++ {
		nodeConfigs = append(nodeConfigs, fmt.Sprintf(nodeConfigPathTmpl, i))
	}

	conf := configuration.BaseCloudConfig{
		Log:             logConfig,
		NodeConfigPaths: nodeConfigs,
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
