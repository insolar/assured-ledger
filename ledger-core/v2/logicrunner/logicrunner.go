// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// Package logicrunner - infrastructure for executing smartcontracts
package logicrunner

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.opencensus.io/stats"

	"github.com/insolar/assured-ledger/ledger-core/v2/configuration"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/bus"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/flow/dispatcher"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	insolarPulse "github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/instracer"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/global"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/artifacts"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/builtin"
	lrCommon "github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/common"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/machinesmanager"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/metrics"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/s_artifact"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/s_contract_requester"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/s_contract_runner"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/s_jet_storage"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/s_sender"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/shutdown"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/sm_object"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/statemachine"
)

// LogicRunner is a general interface of contract executor
type LogicRunner struct {
	ContractRequester insolar.ContractRequester `inject:""`
	PulseAccessor     insolarPulse.Accessor     `inject:""`
	ArtifactManager   artifacts.Client          `inject:""`
	JetStorage        jet.Storage               `inject:""`

	MachinesManager machinesmanager.MachinesManager
	Sender          bus.Sender
	FlowDispatcher  dispatcher.Dispatcher
	ShutdownFlag    shutdown.Flag
	ContractRunner  s_contract_runner.ContractRunnerService

	Conveyor       *conveyor.PulseConveyor
	ConveyorWorker lrCommon.ConveyorWorker

	ObjectCatalog            *sm_object.LocalObjectCatalog
	ArtifactClientService    *s_artifact.ArtifactClientServiceAdapter
	ContractRequesterService *s_contract_requester.ContractRequesterServiceAdapter
	ContractRunnerService    *s_contract_runner.ContractRunnerServiceAdapter
	SenderService            *s_sender.SenderServiceAdapter
	JetStorageService        *s_jet_storage.JetStorageServiceAdapter

	Cfg *configuration.LogicRunner

	rpc *lrCommon.RPC
}

// NewLogicRunner is constructor for LogicRunner
func NewLogicRunner(cfg *configuration.LogicRunner, sender bus.Sender) (*LogicRunner, error) {
	if cfg == nil {
		return nil, errors.New("LogicRunner have nil configuration")
	}
	res := LogicRunner{
		Cfg:    cfg,
		Sender: sender,
	}

	return &res, nil
}

func (lr *LogicRunner) LRI() {}

func (lr *LogicRunner) Init(ctx context.Context) error {
	lr.ShutdownFlag = shutdown.NewFlag()
	global.Error("MachinesManager: %v", lr.MachinesManager)
	if lr.MachinesManager == machinesmanager.MachinesManager(nil) {
		global.Error("New MachinesManager")
		lr.MachinesManager = machinesmanager.NewMachinesManager()
	}
	lr.ContractRunner = s_contract_runner.CreateContractRunner(lr.MachinesManager, lr.ArtifactManager)
	lr.rpc = lrCommon.NewRPC(lr.ContractRunner, lr.Cfg)

	// configuration steps for slot machine
	machineConfig := smachine.SlotMachineConfig{
		PollingPeriod:     500 * time.Millisecond,
		PollingTruncate:   1 * time.Millisecond,
		SlotPageSize:      1000,
		ScanCountLimit:    100000,
		SlotMachineLogger: statemachine.ConveyorLoggerFactory{},
	}

	lr.ObjectCatalog = &sm_object.LocalObjectCatalog{}
	lr.ArtifactClientService = s_artifact.CreateArtifactClientService(lr.ArtifactManager)
	lr.ContractRequesterService = s_contract_requester.CreateContractRequesterService(lr.ContractRequester)
	lr.ContractRunnerService = s_contract_runner.CreateContractRunnerService(lr.ContractRunner)
	lr.SenderService = s_sender.CreateSenderService(lr.Sender, lr.PulseAccessor)
	lr.JetStorageService = s_jet_storage.CreateJetStorageService(lr.JetStorage)

	defaultHandlers := statemachine.DefaultHandlersFactory

	lr.Conveyor = conveyor.NewPulseConveyor(context.Background(), conveyor.PulseConveyorConfig{
		ConveyorMachineConfig: machineConfig,
		SlotMachineConfig:     machineConfig,
		EventlessSleep:        100 * time.Millisecond,
		MinCachePulseAge:      100,
		MaxPastPulseAge:       1000,
	}, defaultHandlers, nil)

	lr.Conveyor.AddDependency(lr.ObjectCatalog)
	lr.Conveyor.AddDependency(lr.ArtifactClientService)
	lr.Conveyor.AddDependency(lr.ContractRequesterService)
	lr.Conveyor.AddDependency(lr.ContractRunnerService)
	lr.Conveyor.AddDependency(lr.SenderService)
	lr.Conveyor.AddDependency(lr.JetStorageService)

	lr.ConveyorWorker = lrCommon.NewConveyorWorker()
	lr.ConveyorWorker.AttachTo(lr.Conveyor)

	lr.FlowDispatcher = lrCommon.NewConveyorDispatcher(lr.Conveyor)

	return nil
}

// Start starts logic runner component
func (lr *LogicRunner) Start(ctx context.Context) error {
	if lr.Cfg.RPCListen != "" {
		lr.rpc.Start(ctx)
	}

	bi := builtin.NewBuiltIn(lr.ArtifactManager, lr.ContractRunner)

	err := lr.MachinesManager.RegisterExecutor(insolar.MachineTypeBuiltin, bi)
	if err != nil {
		return err
	}

	// if lr.Cfg.GoPlugin != nil {
	// 	gp := goplugin.NewGoPlugin(lr.Cfg)
	//
	// 	err := lr.MachinesManager.RegisterExecutor(insolar.MachineTypeGoPlugin, gp)
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	lr.ArtifactManager.InjectFinish()

	return nil
}

// Stop stops logic runner component and its executors
func (lr *LogicRunner) Stop(ctx context.Context) error {
	lr.ConveyorWorker.Stop()

	return lr.rpc.Stop(ctx)
}

func (lr *LogicRunner) GracefulStop(ctx context.Context) error {
	waitFunction := lr.ShutdownFlag.Stop(ctx)
	waitFunction()

	return nil
}

func (lr *LogicRunner) OnPulse(ctx context.Context, oldPulse insolar.Pulse, newPulse insolar.Pulse) error {
	onPulseStart := time.Now()
	ctx, span := instracer.StartSpan(ctx, "pulse.logicrunner")
	defer func(ctx context.Context) {
		stats.Record(ctx,
			metrics.LogicRunnerOnPulseTiming.M(float64(time.Since(onPulseStart).Nanoseconds())/1e6))
		span.Finish()
	}(ctx)

	lr.stopIfNeeded(ctx)

	return nil
}

func (lr *LogicRunner) stopIfNeeded(ctx context.Context) {
	lr.ShutdownFlag.Done(ctx, func() bool { return true })
}

func (lr *LogicRunner) sendOnPulseMessagesAsync(ctx context.Context, messages map[insolar.Reference][]payload.Payload) {
	ctx, spanMessages := instracer.StartSpan(ctx, "pulse.logicrunner sending messages")
	spanMessages.SetTag("numMessages", strconv.Itoa(len(messages)))

	var sendWg sync.WaitGroup

	for ref, msg := range messages {
		sendWg.Add(len(msg))
		for _, msg := range msg {
			go lr.sendOnPulseMessage(ctx, ref, msg, &sendWg)
		}
	}

	sendWg.Wait()
	spanMessages.Finish()
}

func (lr *LogicRunner) sendOnPulseMessage(ctx context.Context, objectRef insolar.Reference, payloadObj payload.Payload, sendWg *sync.WaitGroup) {
	defer sendWg.Done()

	msg, err := payload.NewMessage(payloadObj)
	if err != nil {
		inslogger.FromContext(ctx).Error("failed to serialize message: " + err.Error())
		return
	}

	// we dont really care about response, because we are sending this in the beginning of the pulse
	// so flow canceled should not happened, if it does, somebody already restarted
	_, done := lr.Sender.SendRole(ctx, msg, insolar.DynamicRoleVirtualExecutor, objectRef)
	done()
}

func contextWithServiceData(ctx context.Context, data *payload.ServiceData) context.Context {
	ctx = inslogger.ContextWithTrace(ctx, data.LogTraceID)
	ctx = inslogger.WithLoggerLevel(ctx, data.LogLevel)
	if data.TraceSpanData != nil {
		parentSpan := instracer.MustDeserialize(data.TraceSpanData)
		return instracer.WithParentSpan(ctx, parentSpan)
	}
	return ctx
}

func (lr *LogicRunner) AddUnwantedResponse(ctx context.Context, msg insolar.Payload) error {
	return nil
}
