// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/insolar/assured-ledger/ledger-core/application/bootstrap"
	pulsewatcher "github.com/insolar/assured-ledger/ledger-core/cmd/pulsewatcher/config"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/insolar/defaults"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func baseDir() string {
	return defaults.LaunchnetDir()
}

var (
	nodePortsList = []string{
		"127.0.0.1:13831",
		"127.0.0.1:23832",
		"127.0.0.1:33833",
		"127.0.0.1:43834",
		"127.0.0.1:53835",
		"127.0.0.1:53866",
		"127.0.0.1:53877",
		"127.0.0.1:53888",
		"127.0.0.1:53899",
		"127.0.0.1:53100",
		"127.0.0.1:51101",
		"127.0.0.1:53202",
		"127.0.0.1:53404",
		"127.0.0.1:53606",
		"127.0.0.1:53707",
		"127.0.0.1:53808",
		"127.0.0.1:53909",
		"127.0.0.1:51110",
		"127.0.0.1:53111",
		"127.0.0.1:51112",
		"127.0.0.1:53113",
		"127.0.0.1:54114",
		"127.0.0.1:51115",
	}
)

var (
	defaultOutputConfigNameTmpl      = "%d/insolard.yaml"
	defaultHost                      = "127.0.0.1"
	defaultJaegerEndPoint            = ""
	discoveryCertificatePathTemplate = withBaseDir("discoverynodes/certs/discovery_cert_%d.json")
	nodeDataDirectoryTemplate        = "nodes/%d/data"
	nodeCertificatePathTemplate      = "nodes/%d/cert.json"
	pulsewatcherFileName             = withBaseDir("pulsewatcher.yaml")

	prometheusConfigTmpl = "scripts/prom/server.yml.tmpl"
	prometheusFileName   = "prometheus.yaml"

	bootstrapConfigTmpl = "scripts/insolard/bootstrap_template.yaml"
	bootstrapFileName   = withBaseDir("bootstrap.yaml")

	pulsardConfigTmpl = "scripts/insolard/pulsar_template.yaml"
	pulsardFileName   = withBaseDir("pulsar.yaml")

	keeperdConfigTmpl = "scripts/insolard/keeperd_template.yaml"
	keeperdFileName   = withBaseDir("keeperd.yaml")
)

var (
	outputDir  string
	debugLevel string

	numVirtual, numLight, numHeavy int
	base_cloud                     bool
)

func parseInputParams() {
	var rootCmd = &cobra.Command{}

	rootCmd.Flags().BoolVar(
		&base_cloud, "base-cloud", false, "generate base cloud config only")

	rootCmd.Flags().IntVar(
		&numVirtual, "num-virtual-nodes", 5, "number of nodes with role virtual")

	rootCmd.Flags().IntVar(
		&numLight, "num-light-nodes", 0, "number of nodes with role light")

	rootCmd.Flags().IntVar(
		&numHeavy, "num-heavy-nodes", 0, "number of nodes with role heavy")

	rootCmd.Flags().StringVarP(
		&outputDir, "output", "o", baseDir(), "output directory")
	rootCmd.Flags().StringVarP(
		&debugLevel, "debuglevel", "d", "Debug", "debug level")

	err := rootCmd.Execute()
	check("Wrong input params:", err)
}

func writeInsolardConfigs(dir string, insolardConfigs []configuration.Configuration) {
	fmt.Println("generate_insolar_configs.go: writeInsolardConfigs...")
	for index, conf := range insolardConfigs {
		data, err := yaml.Marshal(conf)
		check("Can't Marshal insolard config", err)

		fileName := fmt.Sprintf(defaultOutputConfigNameTmpl, index+1)
		fileName = filepath.Join(dir, fileName)
		err = createFileWithDir(fileName, string(data))
		check("failed to create insolard config: "+fileName, err)
	}
}

func validateNodesConfig() {
	numNodes := numVirtual + numLight + numHeavy

	switch {
	case numNodes == 0:
		check("cannot start network", throw.New("zero node count provided"))
	case numNodes > len(nodePortsList)-1:
		check("cannot start network", throw.New("node number exceeds nodePortsList"))
	}
}

func prepareBootstrapVars() commonConfigVars {
	res := commonConfigVars{
		BaseDir:         baseDir(),
		MajorityRule:    numVirtual + numLight + numHeavy,
		MinHeavyNodes:   numHeavy,
		MinLightNodes:   numLight,
		MinVirtualNodes: numVirtual,
	}

	nodeList := make([]node, 0, numHeavy+numLight+numVirtual)
	for i := 0; i < numHeavy; i++ {
		num := len(nodeList)
		nodeList = append(nodeList, node{
			Addr: nodePortsList[num],
			Role: "heavy",
			Num:  num + 1,
		})
	}

	for i := 0; i < numLight; i++ {
		num := len(nodeList)
		nodeList = append(nodeList, node{
			Addr: nodePortsList[num],
			Role: "light",
			Num:  num + 1,
		})
	}

	for i := 0; i < numVirtual; i++ {
		num := len(nodeList)
		nodeList = append(nodeList, node{
			Addr: nodePortsList[num],
			Role: "virtual",
			Num:  num + 1,
		})
	}

	res.DiscoveryNodes = nodeList

	return res
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

func main() {
	parseInputParams()

	if base_cloud {
		generateBaseCloudConfig()
		return
	}

	validateNodesConfig()

	mustMakeDir(outputDir)

	writeBootstrapConfig()

	bootstrapConf, err := bootstrap.ParseConfig(bootstrapFileName)
	check("Can't read bootstrap config", err)

	pwConfig := pulsewatcher.Config{}
	discoveryNodesConfigs := make([]configuration.Configuration, 0, len(bootstrapConf.DiscoveryNodes))

	promVars := &promConfigVars{
		Jobs: map[string][]string{},
	}

	// process discovery nodes
	for index, node := range bootstrapConf.DiscoveryNodes {
		nodeIndex := index + 1

		conf := newDefaultInsolardConfig()

		conf.Host.Transport.Address = node.Host
		conf.Host.Transport.Protocol = "TCP"

		rpcListenPort := 33300 + (index+nodeIndex)*nodeIndex
		conf.LogicRunner = configuration.NewLogicRunner()
		conf.LogicRunner.GoPlugin.RunnerListen = fmt.Sprintf(defaultHost+":%d", rpcListenPort-1)
		conf.LogicRunner.RPCListen = fmt.Sprintf(defaultHost+":%d", rpcListenPort)

		conf.TestWalletAPI.Address = fmt.Sprintf(defaultHost+":323%02d", nodeIndex)

		conf.APIRunner.Address = fmt.Sprintf(defaultHost+":191%02d", nodeIndex)
		conf.APIRunner.SwaggerPath = "application/api/spec/api-exported.yaml"

		conf.AvailabilityChecker.Enabled = true
		conf.AvailabilityChecker.KeeperURL = "http://127.0.0.1:12012/check"

		conf.AdminAPIRunner.Address = fmt.Sprintf(defaultHost+":190%02d", nodeIndex)
		conf.AdminAPIRunner.SwaggerPath = "application/api/spec/api-exported.yaml"

		conf.Metrics.ListenAddress = fmt.Sprintf(defaultHost+":80%02d", nodeIndex)
		conf.Introspection.Addr = fmt.Sprintf(defaultHost+":555%02d", nodeIndex)

		conf.Tracer.Jaeger.AgentEndpoint = defaultJaegerEndPoint
		conf.Log.Level = debugLevel
		conf.Log.Adapter = "zerolog"
		conf.Log.Formatter = "json"

		conf.KeysPath = bootstrapConf.DiscoveryKeysDir + fmt.Sprintf(bootstrapConf.KeysNameFormat, nodeIndex)
		conf.CertificatePath = fmt.Sprintf(discoveryCertificatePathTemplate, nodeIndex)

		discoveryNodesConfigs = append(discoveryNodesConfigs, conf)

		promVars.addTarget("insolard", conf)

		pwConfig.Nodes = append(pwConfig.Nodes, conf.AdminAPIRunner.Address)
	}

	// process extra nodes
	nodeDataDirectoryTemplate = filepath.Join(outputDir, nodeDataDirectoryTemplate)
	nodeCertificatePathTemplate = filepath.Join(outputDir, nodeCertificatePathTemplate)

	nodesConfigs := make([]configuration.Configuration, 0, len(bootstrapConf.DiscoveryNodes))
	for index, node := range bootstrapConf.Nodes {
		nodeIndex := index + 1

		conf := newDefaultInsolardConfig()

		conf.Host.Transport.Address = node.Host
		conf.Host.Transport.Protocol = "TCP"

		rpcListenPort := 34300 + (index+nodeIndex+len(bootstrapConf.DiscoveryNodes)+1)*nodeIndex
		conf.LogicRunner = configuration.NewLogicRunner()
		conf.LogicRunner.GoPlugin.RunnerListen = fmt.Sprintf(defaultHost+":%d", rpcListenPort-1)
		conf.LogicRunner.RPCListen = fmt.Sprintf(defaultHost+":%d", rpcListenPort)

		conf.APIRunner.Address = fmt.Sprintf(defaultHost+":191%02d", nodeIndex+len(bootstrapConf.DiscoveryNodes))
		conf.AdminAPIRunner.Address = fmt.Sprintf(defaultHost+":190%02d", nodeIndex+len(bootstrapConf.DiscoveryNodes))
		conf.Metrics.ListenAddress = fmt.Sprintf(defaultHost+":80%02d", nodeIndex+len(bootstrapConf.DiscoveryNodes))
		conf.Introspection.Addr = fmt.Sprintf(defaultHost+":555%02d", nodeIndex+len(bootstrapConf.DiscoveryNodes))

		conf.Tracer.Jaeger.AgentEndpoint = defaultJaegerEndPoint
		conf.Log.Level = debugLevel
		conf.Log.Adapter = "zerolog"
		conf.Log.Formatter = "json"

		conf.KeysPath = node.KeysFile
		conf.CertificatePath = fmt.Sprintf(nodeCertificatePathTemplate, nodeIndex)

		nodesConfigs = append(nodesConfigs, conf)

		promVars.addTarget("insolard", conf)

		pwConfig.Nodes = append(pwConfig.Nodes, conf.AdminAPIRunner.Address)
	}

	writePromConfig(promVars)
	writeInsolardConfigs(filepath.Join(outputDir, "/discoverynodes"), discoveryNodesConfigs)
	writeInsolardConfigs(filepath.Join(outputDir, "/nodes"), nodesConfigs)

	pulsarConf := &pulsarConfigVars{}
	pulsarConf.DataDir = withBaseDir("pulsar_data")
	pulsarConf.BaseDir = baseDir()
	for _, node := range bootstrapConf.DiscoveryNodes {
		pulsarConf.BootstrapHosts = append(pulsarConf.BootstrapHosts, node.Host)
	}
	pulsarConf.AgentEndpoint = defaultJaegerEndPoint
	writePulsarConfig(pulsarConf)

	writeKeeperdConfig()

	pwConfig.Interval = 500 * time.Millisecond
	pwConfig.Timeout = 1 * time.Second
	mustMakeDir(filepath.Dir(pulsewatcherFileName))
	err = pulsewatcher.WriteConfig(pulsewatcherFileName, pwConfig)
	check("couldn't write pulsewatcher config file", err)
	fmt.Println("generate_insolar_configs.go: write to file", pulsewatcherFileName)
}

type node struct {
	Addr string
	Role string
	Num  int
}

type commonConfigVars struct {
	BaseDir string

	MajorityRule int

	MinHeavyNodes, MinLightNodes, MinVirtualNodes int

	DiscoveryNodes []node
}

func writeBootstrapConfig() {
	vars := prepareBootstrapVars()

	templates, err := template.ParseFiles(bootstrapConfigTmpl)
	check("Can't parse template: "+bootstrapConfigTmpl, err)

	var b bytes.Buffer
	err = templates.Execute(&b, &vars)
	check("Can't process template: "+bootstrapConfigTmpl, err)

	err = makeFile(bootstrapFileName, b.String())
	check("Can't makeFileWithDir: "+bootstrapFileName, err)
}

var defaultInsloardConf *configuration.Configuration

func newDefaultInsolardConfig() configuration.Configuration {
	if defaultInsloardConf == nil {
		cfg := configuration.NewConfiguration()
		defaultInsloardConf = &cfg
	}

	defaultInsloardConf.Host.PulseWatchdogTimeout = 2592000 // one month
	return *defaultInsloardConf
}

type pulsarConfigVars struct {
	commonConfigVars
	BootstrapHosts []string
	DataDir        string
	AgentEndpoint  string
}

func writePulsarConfig(pcv *pulsarConfigVars) {
	templates, err := template.ParseFiles(pulsardConfigTmpl)
	check("Can't parse template: "+pulsardConfigTmpl, err)

	var b bytes.Buffer
	err = templates.Execute(&b, pcv)
	check("Can't process template: "+pulsardConfigTmpl, err)
	err = makeFile(pulsardFileName, b.String())
	check("Can't makeFileWithDir: "+pulsardFileName, err)
}

func writeKeeperdConfig() {
	templates, err := template.ParseFiles(keeperdConfigTmpl)
	check("Can't parse template: "+keeperdConfigTmpl, err)

	var b bytes.Buffer
	err = templates.Execute(&b, nil)
	check("Can't process template: "+keeperdConfigTmpl, err)
	err = makeFile(keeperdFileName, b.String())
	check("Can't makeFileWithDir: "+keeperdFileName, err)
}

type promConfigVars struct {
	Jobs map[string][]string
}

func (pcv *promConfigVars) addTarget(name string, conf configuration.Configuration) {
	jobs := pcv.Jobs
	addrPair := strings.SplitN(conf.Metrics.ListenAddress, ":", 2)
	addr := "host.docker.internal:" + addrPair[1]
	jobs[name] = append(jobs[name], addr)
}

func writePromConfig(pcv *promConfigVars) {
	templates, err := template.ParseFiles(prometheusConfigTmpl)
	check("Can't parse template: "+prometheusConfigTmpl, err)

	var b bytes.Buffer
	err = templates.Execute(&b, pcv)
	check("Can't process template: "+prometheusConfigTmpl, err)

	err = makeFileWithDir(outputDir, prometheusFileName, b.String())
	check("Can't makeFileWithDir: "+prometheusFileName, err)
}

func makeFile(name string, text string) error {
	fmt.Println("generate_insolar_configs.go: write to file", name)
	return ioutil.WriteFile(name, []byte(text), 0644)
}

func createFileWithDir(file string, text string) error {
	mustMakeDir(filepath.Dir(file))
	return makeFile(file, text)
}

// makeFileWithDir dumps `text` into file named `name` into directory `dir`.
// Creates directory if needed as well as file
func makeFileWithDir(dir string, name string, text string) error {
	err := os.MkdirAll(dir, 0775)
	if err != nil {
		return err
	}
	file := filepath.Join(dir, name)
	return makeFile(file, text)
}

func mustMakeDir(dir string) {
	err := os.MkdirAll(dir, 0775)
	check("couldn't create directory "+dir, err)
	fmt.Println("generate_insolar_configs.go: creates dir", dir)
}

func withBaseDir(subpath string) string {
	return filepath.Join(baseDir(), subpath)
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
