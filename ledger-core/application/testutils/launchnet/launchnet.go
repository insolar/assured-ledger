package launchnet

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/pulsewatcher"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/application/api/requester"
	"github.com/insolar/assured-ledger/ledger-core/insolar/defaults"
)

const (
	HOST            = "http://localhost:"
	AdminPort       = "19002"
	PublicPort      = "19102"
	HostDebug       = "http://localhost:8001"
	TestAdminRPCUrl = "/admin-api/rpc"
)

var (
	AdminHostPort       = HOST + AdminPort
	TestRPCUrl          = HOST + AdminPort + TestAdminRPCUrl
	TestRPCUrlPublic    = HOST + PublicPort + "/api/rpc"
	disableLaunchnet    = false
	testRPCUrlVar       = "INSOLAR_FUNC_RPC_URL"
	testRPCUrlPublicVar = "INSOLAR_FUNC_RPC_URL_PUBLIC"
	TestWalletHost      = "INSOLAR_FUNC_TESTWALLET_HOST"
	keysPathVar         = "INSOLAR_FUNC_KEYS_PATH"

	pulseTimeEnv     = "PULSARD_PULSAR_PULSETIME"
	defaultPulseTime = 5000

	rootOnce    sync.Once
	projectRoot string

	verbose = true
)

func SetVerbose(v bool) {
	verbose = v
}

type OneShotMode int

func (o OneShotMode) ToValue() string {
	switch o {
	case OneShotUndefined:
		panic(throw.IllegalState())
	case OneShotTrue:
		return "TRUE"
	case OneShotFalse:
		return "FALSE"
	default:
		panic(throw.IllegalValue())
	}
}

const (
	OneShotUndefined OneShotMode = iota
	OneShotTrue
	OneShotFalse
)

// rootPath returns project root folder
func rootPath() string {
	rootOnce.Do(func() {
		path, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
		if err != nil {
			panic("failed to get project root")
		}
		projectRoot = strings.TrimSpace(string(path))
	})
	return filepath.Join(projectRoot, "ledger-core")
}

func CustomRunWithPulsar(numVirtual, numLight, numHeavy int, cb func([]string) int) int {
	return customRun(OneShotFalse, numVirtual, numLight, numHeavy, cb)
}

func CustomRunWithoutPulsar(numVirtual, numLight, numHeavy int, cb func([]string) int) int {
	return customRun(OneShotTrue, numVirtual, numLight, numHeavy, cb)
}

func GetPulseTimeEnv() string {
	return os.Getenv(pulseTimeEnv)
}

func GetPulseTime() int {
	if pulseTime := GetPulseTimeEnv(); len(pulseTime) != 0 {
		res, err := strconv.Atoi(pulseTime)
		if err != nil {
			panic(err)
		}
		return res
	}
	return defaultPulseTime
}

func customRun(pulsarOneShot OneShotMode, numVirtual, numLight, numHeavy int, cb func([]string) int) int {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	apiAddresses, teardown, err := newNetSetup(pulsarOneShot, numVirtual, numLight, numHeavy)
	if err != nil {
		fmt.Println("error while setup, skip tests: ", err)
		return 1
	}
	defer teardown()

	go func() {
		sig := <-c
		fmt.Printf("Got %s signal. Aborting...\n", sig)
		teardown()

		os.Exit(2)
	}()

	code := cb(apiAddresses)

	if code != 0 {
		pulsewatcher.OneShot(apiAddresses)
	}
	return code
}

type User struct {
	Ref              string
	PrivKey          string
	PubKey           string
	MigrationAddress string
}

// Path builds a path from either INSOLAR_FUNC_KEYS_PATH or LAUNCHNET_BASE_DIR
func Path(a ...string) string { // nolint:unparam
	// Path set in Enviroment
	keysPath := os.Getenv(keysPathVar)
	if keysPath != "" {
		p := []string{keysPath}
		p = append(p, a[len(a)-1])
		return filepath.Join(p...)
	}
	d := defaults.LaunchnetDir()
	var parts []string
	if strings.HasPrefix(d, "/") {
		parts = []string{d}
	} else {
		parts = []string{rootPath(), d}
	}

	parts = append(parts, a...)
	return filepath.Join(parts...)
}

func GetDiscoveryNodesCount() (int, error) {
	type nodesConf struct {
		DiscoverNodes []interface{} `yaml:"discovery_nodes"`
	}

	var conf nodesConf

	buff, err := ioutil.ReadFile(Path("bootstrap.yaml"))
	if err != nil {
		return 0, throw.W(err, "[ getNumberNodes ] Can't read bootstrap config")
	}

	err = yaml.Unmarshal(buff, &conf)
	if err != nil {
		return 0, throw.W(err, "[ getNumberNodes ] Can't parse bootstrap config")
	}

	return len(conf.DiscoverNodes), nil
}

func GetNodesCount() (int, error) {
	type nodesConf struct {
		DiscoverNodes []interface{} `yaml:"discovery_nodes"`
		Nodes         []interface{} `yaml:"nodes"`
	}

	var conf nodesConf

	buff, err := ioutil.ReadFile(Path("bootstrap.yaml"))
	if err != nil {
		return 0, throw.W(err, "[ getNumberNodes ] Can't read bootstrap config")
	}

	err = yaml.Unmarshal(buff, &conf)
	if err != nil {
		return 0, throw.W(err, "[ getNumberNodes ] Can't parse bootstrap config")
	}

	return len(conf.DiscoverNodes) + len(conf.Nodes), nil
}

func stopInsolard(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	err := cmd.Process.Signal(syscall.SIGHUP)
	if err != nil {
		return throw.W(err, "[ stopInsolard ] failed to kill process:")
	}

	pState, err := cmd.Process.Wait()
	if err != nil {
		return throw.W(err, "[ stopInsolard ] failed to wait process:")
	}

	fmt.Println("[ stopInsolard ] State: ", pState.String())

	return nil
}

type outputWriter struct {
	skip bool
}

func (ow outputWriter) log(a ...interface{}) {
	if !ow.skip {
		fmt.Println(a...)
	}
}

func waitForNetworkState(cfg appConfig, state network.State) error {
	numAttempts := 270
	numNodes := len(cfg.Nodes)
	currentOk := 0

	output := outputWriter{skip: !verbose}
	output.log("Waiting for Network state: ", state.String())

	for i := 0; i < numAttempts; i++ {
		currentOk = 0
		for _, node := range cfg.Nodes {
			resp, err := requester.Status(fmt.Sprintf("http://%s%s", node.AdminAPIRunner.Address, TestAdminRPCUrl))
			if err != nil {
				output.log("[ waitForNet ] Problem with node " + node.AdminAPIRunner.Address + ". Err: " + err.Error())
				break
			}
			if resp.NetworkState != state.String() {
				output.log("[ waitForNet ] Good response from node " + node.AdminAPIRunner.Address + ". Net is not ready. Response: " + resp.NetworkState)
				break
			}
			output.log("[ waitForNet ] Good response from node " + node.AdminAPIRunner.Address + ". Net is ready. Response: " + resp.NetworkState)
			currentOk++
		}
		if currentOk == numNodes {
			break
		}

		time.Sleep(time.Second)
		output.log(fmt.Sprintf("[ waitForNet ] Waiting for net: attempt %d/%d, numOKs: %d\n", i, numAttempts, currentOk))
	}

	if currentOk != numNodes {
		return errors.New("[ waitForNet ] Can't Start net: No attempts left")
	}
	output.log("All nodes have state", state.String())

	return nil
}

func runPulsar(oneShot OneShotMode) error {
	pulsarCmd := exec.Command("sh", "-c", "./bin/pulsard --config .artifacts/launchnet/pulsar.yaml")

parentSwitch:
	switch oneShot {
	case OneShotUndefined:
		pulsarOneshot := os.Getenv("PULSARD_ONESHOT")

		switch pulsarOneshot {
		case "FALSE":
			oneShot = OneShotFalse
		case "TRUE":
			oneShot = OneShotTrue
		default:
			break parentSwitch
		}

		fallthrough
	case OneShotTrue, OneShotFalse:
		pulsarCmd.Env = append(pulsarCmd.Env, fmt.Sprintf("PULSARD_ONESHOT=%s", oneShot.ToValue()))

	default:
		panic(throw.IllegalValue())
	}

	pulsarCmd.Env = append(pulsarCmd.Env, fmt.Sprintf("%s=%d", pulseTimeEnv, GetPulseTime()))

	if err := pulsarCmd.Start(); err != nil {
		return throw.W(err, "failed to launch pulsar")
	}

	if verbose {
		fmt.Println("Pulsar launched")
	}
	return nil
}

func waitForNet(cfg appConfig, oneShot OneShotMode) error {
	err := waitForNetworkState(cfg, network.WaitPulsar)
	if err != nil {
		return throw.W(err, "Can't wait for NetworkState "+network.WaitPulsar.String())
	}

	err = runPulsar(oneShot)
	if err != nil {
		return throw.W(err, "Can't run pulsar")
	}

	err = waitForNetworkState(cfg, network.CompleteNetworkState)
	if err != nil {
		return throw.W(err, "Can't wait for NetworkState "+network.CompleteNetworkState.String())
	}

	return nil
}

func startCustomNet(pulsarOneShot OneShotMode, numVirtual, numLight, numHeavy int) (*exec.Cmd, []string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, nil, throw.W(err, "failed to get working directory")
	}
	rootPath := rootPath()

	err = os.Chdir(rootPath)
	if err != nil {
		return nil, nil, throw.W(err, "[ startNet  ] Can't change dir")
	}
	defer func() {
		_ = os.Chdir(cwd)
	}()

	cmd := exec.Command("./scripts/insolard/launchnet.sh", "-pwg")
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("NUM_DISCOVERY_VIRTUAL_NODES=%d", numVirtual))
	cmd.Env = append(cmd.Env, fmt.Sprintf("NUM_DISCOVERY_LIGHT_NODES=%d", numLight))
	cmd.Env = append(cmd.Env, fmt.Sprintf("NUM_DISCOVERY_HEAVY_NODES=%d", numHeavy))

	err = waitForLaunch(cmd)
	if err != nil {
		return cmd, nil, throw.W(err, "[ startNet ] couldn't waitForLaunch more")
	}

	appCfg, err := readAppConfig()
	if err != nil {
		return cmd, nil, throw.W(err, "[ startNet ] couldn't read nodes config")
	}

	err = waitForNet(appCfg, pulsarOneShot)
	if err != nil {
		return cmd, nil, throw.W(err, "[ startNet ] couldn't waitForNet more")
	}

	apiAddresses := make([]string, 0, len(appCfg.Nodes))
	for _, nodeCfg := range appCfg.Nodes {
		apiAddresses = append(apiAddresses, nodeCfg.TestWalletAPI.Address)
	}

	return cmd, apiAddresses, nil
}

func startNet() (*exec.Cmd, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, throw.W(err, "failed to get working directory")
	}
	rootPath := rootPath()

	err = os.Chdir(rootPath)
	if err != nil {
		return nil, throw.W(err, "[ startNet  ] Can't change dir")
	}
	defer func() {
		_ = os.Chdir(cwd)
	}()

	cmd := exec.Command("./scripts/insolard/launchnet.sh", "-pwdg")
	err = waitForLaunch(cmd)
	if err != nil {
		return cmd, throw.W(err, "[ startNet ] couldn't waitForLaunch more")
	}

	appCfg, err := readAppConfig()
	if err != nil {
		return cmd, throw.W(err, "[ startNet ] couldn't read nodes config")
	}

	err = waitForNet(appCfg, OneShotUndefined)
	if err != nil {
		return cmd, throw.W(err, "[ startNet ] couldn't waitForNet more")
	}

	return cmd, nil
}

type nodeConfig struct {
	AdminAPIRunner configuration.APIRunner
	TestWalletAPI  configuration.TestWalletAPI
}

type appConfig struct {
	Nodes []nodeConfig
}

func readAppConfig() (appConfig, error) {
	res := appConfig{}
	discoverNodes, err := GetDiscoveryNodesCount()
	if err != nil {
		return res, throw.W(err, "failed to get discovery nodes number")
	}

	res.Nodes = make([]nodeConfig, 0, discoverNodes)
	for i := 1; i <= discoverNodes; i++ {
		nodeCfg, err := readNodeConfig(fmt.Sprintf("discoverynodes/%d/insolard.yaml", i))
		if err != nil {
			return res, throw.W(err, "failed to get discovery node config")
		}
		res.Nodes = append(res.Nodes, nodeCfg)
	}

	return res, nil
}

func readNodeConfig(path string) (nodeConfig, error) {
	var conf nodeConfig
	buff, err := ioutil.ReadFile(Path(path))
	if err != nil {
		return conf, throw.W(err, "[ getNumberNodes ] Can't read bootstrap config")
	}

	err = yaml.Unmarshal(buff, &conf)
	if err != nil {
		return conf, throw.W(err, "[ getNumberNodes ] Can't parse bootstrap config")
	}

	return conf, nil
}

var logRotatorEnableVar = "LOGROTATOR_ENABLE"

// LogRotateEnabled checks is log rotation enabled by environment variable.
func LogRotateEnabled() bool {
	return os.Getenv(logRotatorEnableVar) == "1"
}

func waitForLaunch(cmd *exec.Cmd) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return throw.W(err, "[ startNet] could't set stderr: ")
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return throw.W(err, "[ startNet] could't set stderr: ")
	}

	err = cmd.Start()
	if err != nil {
		return throw.W(err, "[ startNet ] Can't run cmd")
	}

	done := make(chan bool, 1)
	timeout := 240 * time.Second

	output := outputWriter{skip: !verbose}
	go func() {
		scanner := bufio.NewScanner(stdout)
		output.log("Insolard output: ")
		for scanner.Scan() {
			line := scanner.Text()
			output.log(line)
			if strings.Contains(line, "start discovery nodes ...") {
				done <- true
			}
		}
	}()
	if verbose {
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				line := scanner.Text()
				fmt.Println(line)
			}
		}()
	}

	cmdCompleted := make(chan error, 1)
	go func() {
		cmdCompleted <- cmd.Wait()
	}()
	select {
	case err := <-cmdCompleted:
		cmdCompleted <- nil
		return throw.New("[ waitForLaunch ] insolard finished unexpectedly: " + err.Error())
	case <-done:
		return nil
	case <-time.After(timeout):
		return throw.New("[ waitForLaunch ] could't wait for launch: timeout of %s was exceeded", timeout)
	}
}

func RunOnlyWithLaunchnet(t *testing.T) {
	if disableLaunchnet {
		t.Skip()
	}
}

func newNetSetup(pulsarOneShot OneShotMode, numVirtual, numLight, numHeavy int) (apiAddresses []string, cancelFunc func(), err error) {
	cmd, apiAddresses, err := startCustomNet(pulsarOneShot, numVirtual, numLight, numHeavy)
	cancelFunc = func() {}
	if cmd != nil {
		cancelFunc = func() {
			err := stopInsolard(cmd)
			if err != nil {
				fmt.Println("[ teardown ]  failed to stop insolard:", err)
				return
			}
			fmt.Println("[ teardown ] insolard was successfully stopped")
		}
	}
	if err != nil {
		return nil, cancelFunc, throw.W(err, "[ setup ] could't startNet")
	}

	return apiAddresses, cancelFunc, nil
}

func setup() (cancelFunc func(), err error) {
	testRPCUrl := os.Getenv(testRPCUrlVar)
	testRPCUrlPublic := os.Getenv(testRPCUrlPublicVar)

	externalLaunchnet := testRPCUrl != "" && testRPCUrlPublic != ""
	if externalLaunchnet {
		TestRPCUrl = testRPCUrl
		TestRPCUrlPublic = testRPCUrlPublic
		url := strings.Split(TestRPCUrlPublic, "/")
		AdminHostPort = strings.Join(url[0:len(url)-1], "/")
		disableLaunchnet = true

		return func() {}, nil
	}

	cmd, err := startNet()
	cancelFunc = func() {}
	if cmd != nil {
		cancelFunc = func() {
			err := stopInsolard(cmd)
			if err != nil {
				fmt.Println("[ teardown ]  failed to stop insolard:", err)
				return
			}
			fmt.Println("[ teardown ] insolard was successfully stopped")
		}
	}
	if err != nil {
		return cancelFunc, throw.W(err, "[ setup ] could't startNet")
	}

	return cancelFunc, nil
}

func pulseWatcherPath() (string, string) {
	insDir := defaults.RootModuleDir()
	pulseWatcher := filepath.Join(insDir, "bin", "pulsewatcher")

	baseDir := defaults.PathWithBaseDir(defaults.LaunchnetDir(), insDir)
	config := filepath.Join(baseDir, "pulsewatcher.yaml")
	return pulseWatcher, config
}

// RotateLogs rotates launchnet logs, verbose flag enables printing what happens.
func RotateLogs(verbose bool) {
	launchnetDir := defaults.PathWithBaseDir(defaults.LaunchnetDir(), defaults.RootModuleDir())
	dirPattern := filepath.Join(launchnetDir, "logs/*/*/*.log")

	rmCmd := "rm -vf " + dirPattern
	cmd := exec.Command("sh", "-c", rmCmd)
	out, err := cmd.Output()
	if err != nil {
		log.Fatal("RotateLogs: failed to execute shell command: ", rmCmd)
	}
	if verbose {
		fmt.Println("RotateLogs removed files:\n", string(out))
	}

	rotateCmd := "killall -v -SIGUSR2 inslogrotator"
	cmd = exec.Command("sh", "-c", rotateCmd)
	out, err = cmd.Output()
	if err != nil {
		if verbose {
			println("RotateLogs killall output:", string(out))
		}
		log.Fatal("RotateLogs: failed to execute shell command:", rotateCmd)
	}
}

var dumpMetricsEnabledVar = "DUMP_METRICS_ENABLE"

// LogRotateEnabled checks is log rotation enabled by environment variable.
func DumpMetricsEnabled() bool {
	return os.Getenv(dumpMetricsEnabledVar) == "1"
}

// FetchAndSaveMetrics fetches all nodes metric endpoints and saves result to files in
// logs/metrics/$iteration/<node-addr>.txt files.
func FetchAndSaveMetrics(iteration int) ([][]byte, error) {
	n, err := GetNodesCount()
	if err != nil {
		return nil, err
	}
	addrs := make([]string, n)
	for i := 0; i < n; i++ {
		addrs[i] = fmt.Sprintf(HOST+"80%02d", i+1)
	}
	results := make([][]byte, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i, addr := range addrs {
		i := i
		addr := addr + "/metrics"
		go func() {
			defer wg.Done()

			r, err := fetchMetrics(addr)
			if err != nil {
				fetchErr := fmt.Sprintf("%v fetch failed: %v\n", addr, err.Error())
				results[i] = []byte(fetchErr)
				return
			}
			results[i] = r
		}()
	}
	wg.Wait()

	insDir := defaults.RootModuleDir()
	subDir := fmt.Sprintf("%04d", iteration)
	outDir := filepath.Join(insDir, defaults.LaunchnetDir(), "logs/metrics", subDir)
	if err := os.MkdirAll(outDir, os.ModePerm); err != nil {
		return nil, throw.W(err, "failed to create metrics subdirectory")
	}

	for i, b := range results {
		outFile := addrs[i][strings.Index(addrs[i], "://")+3:]
		outFile = strings.ReplaceAll(outFile, ":", "-")
		outFile = filepath.Join(outDir, outFile) + ".txt"

		err := ioutil.WriteFile(outFile, b, 0640)
		if err != nil {
			return nil, throw.W(err, "write metrics failed")
		}
		fmt.Printf("Dump metrics from %v to %v\n", addrs[i], outFile)
	}
	return results, nil
}

func fetchMetrics(fetchURL string) ([]byte, error) {
	r, err := http.Get(fetchURL)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Failed to fetch metrics, got %v code", r.StatusCode)
	}
	return ioutil.ReadAll(r.Body)
}
