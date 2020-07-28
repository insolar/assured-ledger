// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package launchnet

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/application/api/requester"
	"github.com/insolar/assured-ledger/ledger-core/insolar/defaults"
)

const HOST = "http://localhost:"
const AdminPort = "19002"
const PublicPort = "19102"
const HostDebug = "http://localhost:8001"
const TestAdminRPCUrl = "/admin-api/rpc"

var AdminHostPort = HOST + AdminPort
var TestRPCUrl = HOST + AdminPort + TestAdminRPCUrl
var TestRPCUrlPublic = HOST + PublicPort + "/api/rpc"
var disableLaunchnet = false
var testRPCUrlVar = "INSOLAR_FUNC_RPC_URL"
var testRPCUrlPublicVar = "INSOLAR_FUNC_RPC_URL_PUBLIC"
var TestWalletHost = "INSOLAR_FUNC_TESTWALLET_HOST"
var keysPathVar = "INSOLAR_FUNC_KEYS_PATH"

var cmd *exec.Cmd
var cmdCompleted = make(chan error, 1)
var stdin io.WriteCloser
var stdout io.ReadCloser
var stderr io.ReadCloser

var projectRoot string
var rootOnce sync.Once

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

// Method starts launchnet before execution of callback function (cb) and stops launchnet after.
// Returns exit code as a result from calling callback function.
func Run(cb func() int) int {
	err := setup()
	defer teardown()
	if err != nil {
		fmt.Println("error while setup, skip tests: ", err)
		return 1
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)

	go func() {
		sig := <-c
		fmt.Printf("Got %s signal. Aborting...\n", sig)
		teardown()

		os.Exit(2)
	}()

	pulseWatcher, config := pulseWatcherPath()

	code := cb()

	if code != 0 {
		out, err := exec.Command(pulseWatcher, "-c", config, "-s").CombinedOutput()
		if err != nil {
			fmt.Println("PulseWatcher execution error: ", err)
			return 1
		}
		fmt.Println(string(out))
	}
	return code
}

type User struct {
	Ref              string
	PrivKey          string
	PubKey           string
	MigrationAddress string
}

// launchnetPath builds a path from either INSOLAR_FUNC_KEYS_PATH or LAUNCHNET_BASE_DIR
func launchnetPath(a ...string) (string, error) { // nolint:unparam
	// Path set in Enviroment
	keysPath := os.Getenv(keysPathVar)
	if keysPath != "" {
		p := []string{keysPath}
		p = append(p, a[len(a)-1])
		return filepath.Join(p...), nil
	}
	d := defaults.LaunchnetDir()
	var parts []string
	if strings.HasPrefix(d, "/") {
		parts = []string{d}
	} else {
		parts = []string{rootPath(), d}
	}

	parts = append(parts, a...)
	return filepath.Join(parts...), nil
}

func GetNodesCount() (int, error) {
	type nodesConf struct {
		DiscoverNodes []interface{} `yaml:"discovery_nodes"`
		Nodes         []interface{} `yaml:"nodes"`
	}

	var conf nodesConf

	path, err := launchnetPath("bootstrap.yaml")
	if err != nil {
		return 0, err
	}
	buff, err := ioutil.ReadFile(path)
	if err != nil {
		return 0, errors.W(err, "[ getNumberNodes ] Can't read bootstrap config")
	}

	err = yaml.Unmarshal(buff, &conf)
	if err != nil {
		return 0, errors.W(err, "[ getNumberNodes ] Can't parse bootstrap config")
	}

	return len(conf.DiscoverNodes) + len(conf.Nodes), nil
}

func stopInsolard() error {
	if stdin != nil {
		defer stdin.Close()
	}
	if stdout != nil {
		defer stdout.Close()
	}

	if cmd == nil || cmd.Process == nil {
		return nil
	}

	err := cmd.Process.Signal(syscall.SIGHUP)
	if err != nil {
		return errors.W(err, "[ stopInsolard ] failed to kill process:")
	}

	pState, err := cmd.Process.Wait()
	if err != nil {
		return errors.W(err, "[ stopInsolard ] failed to wait process:")
	}

	fmt.Println("[ stopInsolard ] State: ", pState.String())

	return nil
}

func waitForNetworkState(state nodeinfo.NetworkState) error {
	numAttempts := 270
	// TODO: read ports from bootstrap config
	ports := []string{
		"19001",
		"19002",
		"19003",
		"19004",
		"19005",
		// "19106",
		// "19107",
		// "19108",
		// "19109",
		// "19110",
		// "19111",
	}
	numNodes := len(ports)
	currentOk := 0
	fmt.Println("Waiting for Network state: ", state.String())
	for i := 0; i < numAttempts; i++ {
		currentOk = 0
		for _, port := range ports {
			resp, err := requester.Status(fmt.Sprintf("%s%s%s", HOST, port, TestAdminRPCUrl))
			if err != nil {
				fmt.Println("[ waitForNet ] Problem with port " + port + ". Err: " + err.Error())
				break
			}
			if resp.NetworkState != state.String() {
				fmt.Println("[ waitForNet ] Good response from port " + port + ". Net is not ready. Response: " + resp.NetworkState)
				break
			}
			fmt.Println("[ waitForNet ] Good response from port " + port + ". Net is ready. Response: " + resp.NetworkState)
			currentOk++
		}
		if currentOk == numNodes {
			break
		}

		time.Sleep(time.Second)
		fmt.Printf("[ waitForNet ] Waiting for net: attempt %d/%d\n", i, numAttempts)
	}

	if currentOk != numNodes {
		return errors.New("[ waitForNet ] Can't Start net: No attempts left")
	}
	fmt.Println("All nodes have state", state.String())

	return nil
}

func runPulsar() error {
	pulsarCmd := exec.Command("sh", "-c", "./bin/pulsard -c .artifacts/launchnet/pulsar.yaml")
	if err := pulsarCmd.Start(); err != nil {
		return errors.W(err, "failed to launch pulsar")
	}
	fmt.Println("Pulsar launched")
	return nil
}

func waitForNet() error {
	err := waitForNetworkState(nodeinfo.WaitPulsar)
	if err != nil {
		return errors.W(err, "Can't wait for NetworkState "+nodeinfo.WaitPulsar.String())
	}

	err = runPulsar()
	if err != nil {
		return errors.W(err, "Can't run pulsar")
	}

	err = waitForNetworkState(nodeinfo.CompleteNetworkState)
	if err != nil {
		return errors.W(err, "Can't wait for NetworkState "+nodeinfo.CompleteNetworkState.String())
	}

	return nil
}

func startNet() error {
	cwd, err := os.Getwd()
	if err != nil {
		return errors.W(err, "failed to get working directory")
	}
	rootPath := rootPath()

	err = os.Chdir(rootPath)
	if err != nil {
		return errors.W(err, "[ startNet  ] Can't change dir")
	}
	defer func() {
		_ = os.Chdir(cwd)
	}()

	cmd = exec.Command("./scripts/insolard/launchnet.sh", "-pwdg")
	stdout, _ = cmd.StdoutPipe()

	stderr, err = cmd.StderrPipe()
	if err != nil {
		return errors.W(err, "[ startNet] could't set stderr: ")
	}

	err = cmd.Start()
	if err != nil {
		return errors.W(err, "[ startNet ] Can't run cmd")
	}

	err = waitForLaunch()
	if err != nil {
		return errors.W(err, "[ startNet ] couldn't waitForLaunch more")
	}

	err = waitForNet()
	if err != nil {
		return errors.W(err, "[ startNet ] couldn't waitForNet more")
	}

	return nil

}

var logRotatorEnableVar = "LOGROTATOR_ENABLE"

// LogRotateEnabled checks is log rotation enabled by environment variable.
func LogRotateEnabled() bool {
	return os.Getenv(logRotatorEnableVar) == "1"
}

func waitForLaunch() error {
	done := make(chan bool, 1)
	timeout := 240 * time.Second

	go func() {
		scanner := bufio.NewScanner(stdout)
		fmt.Println("Insolard output: ")
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line)
			if strings.Contains(line, "start discovery nodes ...") {
				done <- true
			}
		}
	}()
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line)
		}
	}()

	go func() { cmdCompleted <- cmd.Wait() }()
	select {
	case err := <-cmdCompleted:
		cmdCompleted <- nil
		return errors.New("[ waitForLaunch ] insolard finished unexpectedly: " + err.Error())
	case <-done:
		return nil
	case <-time.After(timeout):
		return errors.Errorf("[ waitForLaunch ] could't wait for launch: timeout of %s was exceeded", timeout)
	}
}

func RunOnlyWithLaunchnet(t *testing.T) {
	if disableLaunchnet {
		t.Skip()
	}
}

func setup() error {
	testRPCUrl := os.Getenv(testRPCUrlVar)
	testRPCUrlPublic := os.Getenv(testRPCUrlPublicVar)

	if testRPCUrl == "" || testRPCUrlPublic == "" {
		err := startNet()
		if err != nil {
			return errors.W(err, "[ setup ] could't startNet")
		}
	} else {
		TestRPCUrl = testRPCUrl
		TestRPCUrlPublic = testRPCUrlPublic
		url := strings.Split(TestRPCUrlPublic, "/")
		AdminHostPort = strings.Join(url[0:len(url)-1], "/")
		disableLaunchnet = true
	}
	return nil
}

func pulseWatcherPath() (string, string) {
	insDir := defaults.RootModuleDir()
	pulseWatcher := filepath.Join(insDir, "bin", "pulsewatcher")

	baseDir := defaults.PathWithBaseDir(defaults.LaunchnetDir(), insDir)
	config := filepath.Join(baseDir, "pulsewatcher.yaml")
	return pulseWatcher, config
}

func teardown() {
	err := stopInsolard()
	if err != nil {
		fmt.Println("[ teardown ]  failed to stop insolard:", err)
		return
	}
	fmt.Println("[ teardown ] insolard was successfully stopped")
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
		return nil, errors.W(err, "failed to create metrics subdirectory")
	}

	for i, b := range results {
		outFile := addrs[i][strings.Index(addrs[i], "://")+3:]
		outFile = strings.ReplaceAll(outFile, ":", "-")
		outFile = filepath.Join(outDir, outFile) + ".txt"

		err := ioutil.WriteFile(outFile, b, 0640)
		if err != nil {
			return nil, errors.W(err, "write metrics failed")
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
