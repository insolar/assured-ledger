package launchnet

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/pulsewatcher"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/server"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/cloud"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type cloudOption func(runner *CloudRunner)

type PulsarMode uint8

const (
	RegularPulsar PulsarMode = iota
	ManualPulsar
)

func WithMinRoles(virtual, light, heavy uint) func(runner *CloudRunner) {
	return func(runner *CloudRunner) {
		runner.MinRoles = cloud.NodeConfiguration{
			Virtual:       virtual,
			LightMaterial: light,
			HeavyMaterial: heavy,
		}
	}
}

func WithPrepared(virtual, light, heavy uint) func(runner *CloudRunner) {
	return func(runner *CloudRunner) {
		runner.Prepared = cloud.NodeConfiguration{
			Virtual:       virtual,
			LightMaterial: light,
			HeavyMaterial: heavy,
		}
	}
}

func WithRunning(virtual, light, heavy uint) func(runner *CloudRunner) {
	return func(runner *CloudRunner) {
		runner.Running = cloud.NodeConfiguration{
			Virtual:       virtual,
			LightMaterial: light,
			HeavyMaterial: heavy,
		}
	}
}

func WithMajorityRule(num int) func(runner *CloudRunner) {
	return func(runner *CloudRunner) {
		runner.majorityRule = num
	}
}

func WithDefaultLogLevel(level log.Level) func(runner *CloudRunner) {
	return func(runner *CloudRunner) {
		runner.defaultLogLevel = level
	}
}

func WithPulsarMode(mode PulsarMode) func(runner *CloudRunner) {
	return func(runner *CloudRunner) {
		runner.pulsarMode = mode
	}
}

func WithCloudFileLogging() func(runner *CloudRunner) {
	return func(runner *CloudRunner) {
		runner.cloudFileLogging = true
	}
}

func PrepareCloudRunner(options ...cloudOption) *CloudRunner {
	cr := CloudRunner{
		defaultLogLevel: log.DebugLevel,
		pulsarMode:      getPulseModeFromEnv(),
	}
	for _, o := range options {
		o(&cr)
	}
	cr.PrepareConfig()
	return &cr
}

type CloudRunner struct {
	MinRoles cloud.NodeConfiguration
	Prepared cloud.NodeConfiguration
	Running  cloud.NodeConfiguration

	majorityRule int

	pulsarMode PulsarMode

	defaultLogLevel log.Level

	cloudFileLogging bool

	ConfProvider *cloud.ConfigurationProvider
}

func (cr *CloudRunner) PrepareConfig() {
	if cr.Running.IsZero() {
		panic("No running nodes")
	}

	if cr.Prepared.IsZero() {
		cr.Prepared = cr.Running
	}

	if cr.MinRoles.IsZero() {
		cr.MinRoles = cr.Running
	}

	if cr.majorityRule == 0 {
		cr.majorityRule = int(cr.Running.Virtual + cr.Running.LightMaterial + cr.Running.HeavyMaterial)
	}

	cloudSettings := cloud.Settings{
		MinRoles: cr.MinRoles,
		Prepared: cr.Prepared,
		Running:  cr.Running,

		MajorityRule:     cr.majorityRule,
		CloudFileLogging: cr.cloudFileLogging,

		Log:    struct{ Level string }{Level: cr.defaultLogLevel.String()},
		Pulsar: struct{ PulseTime int }{PulseTime: GetPulseTime()},
	}

	cr.ConfProvider = cloud.NewConfigurationProvider(cloudSettings)
}

func prepareCloudForOneShotMode(confProvider *cloud.ConfigurationProvider) *insapp.Server {
	controller := cloud.NewController()
	s, _ := server.NewControlledMultiServer(controller, confProvider)
	go func() {
		s.WaitStarted()

		allNodes := make(map[reference.Global]struct{})
		for _, conf := range confProvider.GetAppConfigs() {
			cert, err := confProvider.CertificateFactory(nil, nil, conf.CertificatePath)
			if err != nil {
				panic(err)
			}
			allNodes[cert.GetCertificate().GetNodeRef()] = struct{}{}
		}

		pulseGenerator := testutils.NewPulseGenerator(uint16(confProvider.PulsarConfig.Pulsar.NumberDelta), nil, nil)
		for i := 0; i < 2; i++ {
			_ = pulseGenerator.Generate()
			controller.PartialDistribute(context.Background(), pulseGenerator.GetLastPulsePacket(), allNodes)
		}
	}()

	return s
}

//nolint:goconst
func getPulseModeFromEnv() PulsarMode {
	pulsarOneshot := os.Getenv("PULSARD_ONESHOT")
	switch pulsarOneshot {
	case "TRUE":
		return ManualPulsar
	case "FALSE", "":
		return RegularPulsar
	default:
		panic(throw.IllegalValue())
	}
}

func (cr CloudRunner) SetupCloud() (func(), error) {
	return cr.SetupCloudCustom(cr.pulsarMode)
}

func (cr CloudRunner) SetupCloudCustom(pulsarMode PulsarMode) (func(), error) {
	var s *insapp.Server
	if pulsarMode == ManualPulsar {
		s = prepareCloudForOneShotMode(cr.ConfProvider)
	} else {
		s = server.NewMultiServer(cr.ConfProvider)
	}
	go func() {
		s.Serve()
	}()

	nodes := make([]nodeConfig, 0, len(cr.ConfProvider.GetAppConfigs()))
	for _, appCfg := range cr.ConfProvider.GetAppConfigs() {
		nodes = append(nodes, nodeConfig{
			AdminAPIRunner: appCfg.AdminAPIRunner,
			TestWalletAPI:  appCfg.TestWalletAPI,
		})
	}

	SetVerbose(false)
	err := waitForNetworkState(appConfig{Nodes: nodes}, network.CompleteNetworkState)
	if err != nil {
		return s.Stop, throw.W(err, "Can't wait for NetworkState "+network.CompleteNetworkState.String())
	}
	return s.Stop, nil
}

func (cr *CloudRunner) Run(cb func([]string) int) int {
	teardown, err := cr.SetupCloud()
	defer teardown()
	if err != nil {
		fmt.Println("error while setup, skip tests: ", err)
		return 1
	}

	apiAddresses := make([]string, 0, len(cr.ConfProvider.GetAppConfigs()))
	adminAddresses := make([]string, 0, len(cr.ConfProvider.GetAppConfigs()))
	for _, el := range cr.ConfProvider.GetAppConfigs() {
		apiAddresses = append(apiAddresses, el.TestWalletAPI.Address)
		adminAddresses = append(adminAddresses, el.AdminAPIRunner.Address)
	}

	code := cb(apiAddresses)

	if code != 0 {
		pulsewatcher.OneShot(adminAddresses)
	}
	return code
}

func (cr CloudRunner) SetupControlledRun() (*Helper, func(), error) {
	var s *insapp.Server
	controller := cloud.NewController()
	s, nodeController := server.NewControlledMultiServer(controller, cr.ConfProvider)
	go func() {
		s.Serve()
	}()

	helper := &Helper{
		pulseLock: &sync.Mutex{},
		pulseGenerator: testutils.NewPulseGenerator(
			uint16(cr.ConfProvider.PulsarConfig.Pulsar.NumberDelta), nil, nil),
		NodeController:        nodeController,
		NetworkController:     controller,
		ConfigurationProvider: cr.ConfProvider,
	}

	s.WaitStarted()

	for i := 0; i < 2; i++ {
		helper.IncrementPulse(context.TODO())
	}

	nodes := make([]nodeConfig, 0, len(cr.ConfProvider.GetAppConfigs()))
	for _, appCfg := range cr.ConfProvider.GetAppConfigs() {
		nodes = append(nodes, nodeConfig{
			AdminAPIRunner: appCfg.AdminAPIRunner,
			TestWalletAPI:  appCfg.TestWalletAPI,
		})
	}

	SetVerbose(false)
	err := waitForNetworkState(appConfig{Nodes: nodes}, network.CompleteNetworkState)
	if err != nil {
		return nil, s.Stop, throw.W(err, "Can't wait for NetworkState "+network.CompleteNetworkState.String())
	}
	return helper, s.Stop, nil
}

func (cr *CloudRunner) ControlledRun(cb func(helper *Helper) error) error {
	helper, teardown, err := cr.SetupControlledRun()
	defer teardown()
	if err != nil {
		fmt.Println("error while setup, skip tests: ", err)
		return err
	}

	err = cb(helper)
	if err != nil {
		pulsewatcher.OneShot(helper.GetAdminAddresses())
	}
	return err
}
