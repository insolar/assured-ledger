// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build slowtest

package integration

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"math"
	"math/rand"
	"os"
	"path"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/insolar/component-manager"
	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/api/requester"
	"github.com/insolar/assured-ledger/ledger-core/v2/application/api/seedmanager"
	"github.com/insolar/assured-ledger/ledger-core/v2/application/genesisrefs"
	"github.com/insolar/assured-ledger/ledger-core/v2/configuration"
	"github.com/insolar/assured-ledger/ledger-core/v2/contractrequester"
	"github.com/insolar/assured-ledger/ledger-core/v2/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/bus"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/bus/meta"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/jetcoordinator"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/reply"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/utils"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger/logwatermill"
	"github.com/insolar/assured-ledger/ledger-core/v2/keystore"
	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/artifacts"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/machinesmanager"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/pulsemanager"
	"github.com/insolar/assured-ledger/ledger-core/v2/network"
	"github.com/insolar/assured-ledger/ledger-core/v2/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/integration/mimic"
)

type flowCallbackType func(meta payload.Meta, pl payload.Payload) []payload.Payload

func NodeLight() insolar.Reference {
	return light.ref
}

type Server struct {
	t    testing.TB
	ctx  context.Context
	lock sync.RWMutex
	cfg  configuration.Configuration

	// configuration parameters
	isPrepared          bool
	stubMachinesManager machinesmanager.MachinesManager
	stubFlowCallback    flowCallbackType
	withGenesis         bool

	componentManager  *component.Manager
	stopper           func()
	clientSender      bus.Sender
	logicRunner       *logicrunner.LogicRunner
	contractRequester *contractrequester.ContractRequester

	pulseGenerator *mimic.PulseGenerator
	pulseStorage   *pulse.StorageMem
	pulseManager   insolar.PulseManager

	mimic mimic.Ledger

	ExternalPubSub, IncomingPubSub *gochannel.GoChannel
}

func DefaultVMConfig() configuration.Configuration {
	cfg := configuration.Configuration{}
	cfg.KeysPath = "testdata/bootstrap_keys.json"
	cfg.Ledger.LightChainLimit = math.MaxInt32
	cfg.LogicRunner = configuration.NewLogicRunner()
	cfg.Bus.ReplyTimeout = 5 * time.Second
	cfg.Log = configuration.NewLog()
	cfg.Log.Level = log.InfoLevel.String()            // insolar.DebugLevel.String()
	cfg.Log.Formatter = logcommon.JsonFormat.String() // insolar.TextFormat.String()
	return cfg
}

func checkError(ctx context.Context, err error, message string) {
	if err == nil {
		return
	}
	inslogger.FromContext(ctx).Fatalf("%v: %v", message, err.Error())
}

var verboseWM bool

func init() {
	flag.BoolVar(&verboseWM, "verbose-wm", false, "flag to enable watermill logging")
}

func NewVirtualServer(t testing.TB, ctx context.Context, cfg configuration.Configuration) *Server {
	return &Server{
		t:   t,
		ctx: ctx,
		cfg: cfg,
	}
}

func (s *Server) SetMachinesManager(machinesManager machinesmanager.MachinesManager) *Server {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.isPrepared {
		return nil
	}

	s.stubMachinesManager = machinesManager
	return s
}

func (s *Server) SetLightCallbacks(flowCallback flowCallbackType) *Server {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.isPrepared {
		return nil
	}

	s.stubFlowCallback = flowCallback
	return s
}

func (s *Server) WithGenesis() *Server {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.isPrepared {
		return nil
	}

	s.withGenesis = true
	return s
}

func (s *Server) PrepareAndStart() (*Server, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	ctx, logger := inslogger.InitNodeLogger(s.ctx, s.cfg.Log, "", "virtual")
	s.ctx = ctx

	machinesManager := s.stubMachinesManager
	if machinesManager == machinesmanager.MachinesManager(nil) {
		machinesManager = machinesmanager.NewMachinesManager()
	}

	cm := component.NewManager(nil)

	// Cryptography.
	var (
		KeyProcessor  insolar.KeyProcessor
		CryptoScheme  insolar.PlatformCryptographyScheme
		CryptoService insolar.CryptographyService
		KeyStore      insolar.KeyStore
	)
	{
		var err error
		// Private key storage.
		KeyStore, err = keystore.NewKeyStore(s.cfg.KeysPath)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load KeyStore")
		}
		// Public key manipulations.
		KeyProcessor = platformpolicy.NewKeyProcessor()
		// Platform cryptography.
		CryptoScheme = platformpolicy.NewPlatformCryptographyScheme()
		// Sign, verify, etc.
		CryptoService = cryptography.NewCryptographyService()
	}

	// Network.
	var (
		NodeNetwork network.NodeNetwork
	)
	{
		NodeNetwork = newNodeNetMock(&virtual)
	}

	// Role calculations.
	var (
		Coordinator *jetcoordinator.Coordinator
		Pulses      *pulse.StorageMem
		Jets        *jet.Store
		Nodes       *node.Storage
	)
	{
		Nodes = node.NewStorage()
		Pulses = pulse.NewStorageMem()
		Jets = jet.NewStore()

		Coordinator = jetcoordinator.NewJetCoordinator(s.cfg.Ledger.LightChainLimit, virtual.ref)
		Coordinator.PulseCalculator = Pulses
		Coordinator.PulseAccessor = Pulses
		Coordinator.JetAccessor = Jets
		Coordinator.PlatformCryptographyScheme = CryptoScheme
		Coordinator.Nodes = Nodes
	}

	// PulseManager
	var (
		PulseManager *pulsemanager.PulseManager
	)
	{
		PulseManager = pulsemanager.NewPulseManager()
	}

	wmLogger := watermill.LoggerAdapter(watermill.NopLogger{})

	if verboseWM {
		wmLogger = logwatermill.NewWatermillLogAdapter(logger)
	}

	// Communication.
	var (
		ClientBus                      *bus.Bus
		ExternalPubSub, IncomingPubSub *gochannel.GoChannel
	)
	{
		pubsub := gochannel.NewGoChannel(gochannel.Config{}, wmLogger)
		ExternalPubSub = pubsub
		IncomingPubSub = pubsub

		c := jetcoordinator.NewJetCoordinator(s.cfg.Ledger.LightChainLimit, virtual.ref)
		c.PulseCalculator = Pulses
		c.PulseAccessor = Pulses
		c.JetAccessor = Jets
		c.PlatformCryptographyScheme = CryptoScheme
		c.Nodes = Nodes
		ClientBus = bus.NewBus(s.cfg.Bus, IncomingPubSub, Pulses, c, CryptoScheme)
	}

	logicRunner, err := logicrunner.NewLogicRunner(&s.cfg.LogicRunner, ClientBus)
	checkError(ctx, err, "failed to start LogicRunner")

	contractRequester, err := contractrequester.New(
		ClientBus,
		Pulses,
		Coordinator,
		CryptoScheme,
	)
	checkError(ctx, err, "failed to start ContractRequester")

	// TODO: remove this hack in INS-3341
	contractRequester.LR = logicRunner

	cm.Inject(CryptoScheme,
		KeyStore,
		CryptoService,
		KeyProcessor,
		Coordinator,
		logicRunner,

		ClientBus,
		IncomingPubSub,
		contractRequester,
		artifacts.NewClient(ClientBus),
		Pulses,
		Jets,
		Nodes,

		NodeNetwork,
		PulseManager)

	logicRunner.MachinesManager = machinesManager

	err = cm.Init(ctx)
	checkError(ctx, err, "failed to init components")

	err = cm.Start(ctx)
	checkError(ctx, err, "failed to start components")

	var (
		LedgerMimic mimic.Ledger
	)
	{
		LedgerMimic = mimic.NewMimicLedger(ctx, CryptoScheme, Pulses, Pulses, ClientBus)
	}

	flowCallback := s.stubFlowCallback
	if flowCallback == nil {
		flowCallback = LedgerMimic.ProcessMessage
	}

	// Start routers with handlers.
	outHandler := func(msg *message.Message) error {
		var err error

		if msg.Metadata.Get(meta.Type) == meta.TypeReply {
			err = ExternalPubSub.Publish(getIncomingTopic(msg), msg)
			if err != nil {
				panic(errors.Wrap(err, "failed to publish to self"))
			}
			return nil
		}

		msgMeta := payload.Meta{}
		err = msgMeta.Unmarshal(msg.Payload)
		if err != nil {
			panic(errors.Wrap(err, "failed to unmarshal meta"))
		}

		// Republish as incoming to self.
		if msgMeta.Receiver == virtual.ID() {
			err = ExternalPubSub.Publish(getIncomingTopic(msg), msg)
			if err != nil {
				panic(errors.Wrap(err, "failed to publish to self"))
			}
			return nil
		}

		pl, err := payload.Unmarshal(msgMeta.Payload)
		if err != nil {
			panic(errors.Wrap(err, "failed to unmarshal payload"))
		}
		if msgMeta.Receiver == NodeLight() {
			go func() {
				replies := flowCallback(msgMeta, pl)
				for _, rep := range replies {
					msg, err := payload.NewMessage(rep)
					if err != nil {
						panic(err)
					}
					ClientBus.Reply(context.Background(), msgMeta, msg)
				}
			}()
		}

		clientHandler := func(msg *message.Message) (messages []*message.Message, e error) {
			return nil, nil
		}
		// Republish as incoming to client.
		_, err = ClientBus.IncomingMessageRouter(clientHandler)(msg)

		if err != nil {
			panic(err)
		}
		return nil
	}

	stopper := startWatermill(
		ctx, wmLogger, IncomingPubSub, ClientBus,
		outHandler,
		logicRunner.FlowDispatcher.Process,
		contractRequester.FlowDispatcher.Process,
	)

	PulseManager.AddDispatcher(logicRunner.FlowDispatcher, contractRequester.FlowDispatcher)

	inslogger.FromContext(ctx).WithFields(map[string]interface{}{
		"light":   light.ID().String(),
		"virtual": virtual.ID().String(),
	}).Info("started test server")

	s.componentManager = cm
	s.contractRequester = contractRequester
	s.stopper = stopper
	s.clientSender = ClientBus
	s.mimic = LedgerMimic

	s.pulseManager = PulseManager
	s.pulseStorage = Pulses
	s.pulseGenerator = mimic.NewPulseGenerator(10)
	// s.pulse = *insolar.GenesisPulse

	if s.withGenesis {
		if err := s.LoadGenesis(ctx, ""); err != nil {
			return nil, errors.Wrap(err, "failed to load genesis")
		}

	}

	// First pulse goes in storage then interrupts.
	s.incrementPulse(ctx)
	s.isPrepared = true

	return s, nil
}

func (s *Server) Stop(ctx context.Context) {
	if err := s.componentManager.Stop(ctx); err != nil {
		panic(err)
	}
	s.stopper()
}

func (s *Server) incrementPulse(ctx context.Context) {
	s.pulseGenerator.Generate()

	if err := s.pulseManager.Set(ctx, s.pulseGenerator.GetLastPulseAsPulse()); err != nil {
		panic(err)
	}
}

func (s *Server) IncrementPulse(ctx context.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.incrementPulse(ctx)
}

func (s *Server) SendToSelf(ctx context.Context, pl payload.Payload) (<-chan *message.Message, func()) {
	msg, err := payload.NewMessage(pl)
	if err != nil {
		panic(err)
	}
	msg.Metadata.Set(meta.TraceID, s.t.Name())
	return s.clientSender.SendTarget(ctx, msg, virtual.ID())
}

func startWatermill(
	ctx context.Context,
	logger watermill.LoggerAdapter,
	sub message.Subscriber,
	b *bus.Bus,
	outHandler, inHandler, resultsHandler message.NoPublishHandlerFunc,
) func() {
	inRouter, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		panic(err)
	}
	outRouter, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		panic(err)
	}

	outRouter.AddNoPublisherHandler(
		"OutgoingHandler",
		bus.TopicOutgoing,
		sub,
		outHandler,
	)

	inRouter.AddMiddleware(
		b.IncomingMessageRouter,
	)

	inRouter.AddNoPublisherHandler(
		"IncomingHandler",
		bus.TopicIncoming,
		sub,
		inHandler,
	)

	inRouter.AddNoPublisherHandler(
		"IncomingRequestResultHandler",
		bus.TopicIncomingRequestResults,
		sub,
		resultsHandler)

	startRouter(ctx, inRouter)
	startRouter(ctx, outRouter)

	return stopWatermill(ctx, inRouter, outRouter)
}

func stopWatermill(ctx context.Context, routers ...io.Closer) func() {
	return func() {
		for _, r := range routers {
			err := r.Close()
			if err != nil {
				inslogger.FromContext(ctx).Error("Error while closing router", err)
			}
		}
	}
}

func startRouter(ctx context.Context, router *message.Router) {
	go func() {
		if err := router.Run(ctx); err != nil {
			inslogger.FromContext(ctx).Error("Error while running router", err)
		}
	}()
	<-router.Running()
}

func getIncomingTopic(msg *message.Message) string {
	topic := bus.TopicIncoming
	if msg.Metadata.Get(meta.Type) == meta.TypeReturnResults {
		topic = bus.TopicIncomingRequestResults
	}
	return topic
}

func (s *Server) BasicAPICall(
	ctx context.Context,
	callSite string,
	callParams interface{},
	objectRef insolar.Reference,
	user *User,
) (
	insolar.Reply,
	*insolar.Reference,
	error,
) {

	seed, err := (&seedmanager.SeedGenerator{}).Next()
	if err != nil {
		panic(err.Error())
	}

	privateKey, err := platformpolicy.NewKeyProcessor().ImportPrivateKeyPEM([]byte(user.PrivateKey))
	if err != nil {
		panic(err.Error())
	}

	request := &requester.ContractRequest{
		Request: requester.Request{
			Version: requester.JSONRPCVersion,
			ID:      uint64(rand.Int63()),
			Method:  "contract.call",
		},
		Params: requester.Params{
			Seed:       string(seed[:]),
			CallSite:   callSite,
			CallParams: callParams,
			Reference:  objectRef.String(),
			PublicKey:  user.PublicKey,
			LogLevel:   nil,
			Test:       s.t.Name(),
		},
	}

	jsonRequest, err := json.Marshal(request)
	if err != nil {
		panic(err.Error())
	}

	signature, err := requester.Sign(privateKey, jsonRequest)
	if err != nil {
		panic(err.Error())
	}

	requestArgs, err := insolar.Serialize([]interface{}{jsonRequest, signature})
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to marshal arguments")
	}

	currentPulse, err := s.pulseStorage.Latest(ctx)
	if err != nil {
		panic(err.Error())
	}

	return s.contractRequester.Call(ctx, &objectRef, "Call", []interface{}{requestArgs}, currentPulse.PulseNumber)
}

func (s *Server) LoadGenesis(ctx context.Context, genesisDirectory string) error {
	ctx = inslogger.WithLoggerLevel(ctx, log.ErrorLevel)

	if genesisDirectory == "" {
		genesisDirectory = GenesisDirectory
	}
	return s.mimic.LoadGenesis(ctx, genesisDirectory)
}

type ServerHelper struct {
	s *Server
}

func (h *ServerHelper) createUser(ctx context.Context) (*User, error) {
	ctx, _ = inslogger.WithTraceField(ctx, utils.RandTraceID())

	user, err := NewUserWithKeys()
	if err != nil {
		return nil, errors.Errorf("failed to create new user: " + err.Error())
	}

	{
		callMethodReply, _, err := h.s.BasicAPICall(ctx, "member.create", nil, genesisrefs.ContractRootMember, user)
		if err != nil {
			return nil, errors.Wrap(err, "failed to call member.create")
		}

		var result map[string]interface{}
		if cm, ok := callMethodReply.(*reply.CallMethod); !ok {
			return nil, errors.Wrapf(err, "unexpected type of return value %T", callMethodReply)
		} else if err := insolar.Deserialize(cm.Result, &result); err != nil {
			return nil, errors.Wrap(err, "failed to deserialize result")
		}

		r0, ok := result["Returns"]
		if ok && r0 != nil {
			if r1, ok := r0.([]interface{}); !ok {
				return nil, errors.Errorf("bad response: bad type of 'Returns' [%#v]", r0)
			} else if len(r1) != 2 {
				return nil, errors.Errorf("bad response: bad length of 'Returns' [%#v]", r0)
			} else if r2, ok := r1[0].(map[string]interface{}); !ok {
				return nil, errors.Errorf("bad response: bad type of first value [%#v]", r1)
			} else if r3, ok := r2["reference"]; !ok {
				return nil, errors.Errorf("bad response: absent reference field [%#v]", r2)
			} else if walletReferenceString, ok := r3.(string); !ok {
				return nil, errors.Errorf("bad response: reference field expected to be a string [%#v]", r3)
			} else if walletReference, err := insolar.NewReferenceFromString(walletReferenceString); err != nil {
				return nil, errors.Wrap(err, "bad response: got bad reference")
			} else {
				user.Reference = *walletReference
			}

			return user, nil
		}

		r0, ok = result["Error"]
		if ok && r0 != nil {
			return nil, errors.Errorf("%T: %#v", r0, r0)
		}

		panic("unreachable")
	}
}

func (h *ServerHelper) transferMoney(ctx context.Context, from User, to User, amount int64) (int64, error) {
	ctx, _ = inslogger.WithTraceField(ctx, utils.RandTraceID())

	callParams := map[string]interface{}{
		"amount":            strconv.FormatInt(amount, 10),
		"toMemberReference": to.Reference.String(),
	}
	callMethodReply, _, err := h.s.BasicAPICall(ctx, "member.transfer", callParams, from.Reference, &from)
	if err != nil {
		return 0, errors.Wrap(err, "failed to call member.transfer")
	}

	var result map[string]interface{}
	if cm, ok := callMethodReply.(*reply.CallMethod); !ok {
		return 0, errors.Wrapf(err, "unexpected type of return value %T", callMethodReply)
	} else if err := insolar.Deserialize(cm.Result, &result); err != nil {
		return 0, errors.Wrap(err, "failed to deserialize result")
	}

	r0, ok := result["Returns"]
	if ok && r0 != nil {
		if r1, ok := r0.([]interface{}); !ok {
			return 0, errors.Errorf("bad response: bad type of 'Returns' [%#v]", r0)
		} else if len(r1) != 2 {
			return 0, errors.Errorf("bad response: bad length of 'Returns' [%#v]", r0)
		} else if r2, ok := r1[0].(map[string]interface{}); !ok {
			return 0, errors.Errorf("bad response: bad type of first value [%#v]", r1)
		} else if r3, ok := r2["fee"]; !ok {
			return 0, errors.Errorf("bad response: absent fee field [%#v]", r2)
		} else if feeRaw, ok := r3.(string); !ok {
			return 0, errors.Errorf("bad response: Fee field expected to be a string [%#v]", r3)
		} else if fee, err := strconv.ParseInt(feeRaw, 10, 0); err != nil {
			return 0, errors.Wrapf(err, "failed to parse fee [%#v]", feeRaw)
		} else {
			return fee, nil
		}
	}

	r0, ok = result["Error"]
	if ok && r0 != nil {
		return 0, errors.Errorf("%T: %#v", r0, r0)
	}

	panic("unreachable")
}

func (h *ServerHelper) getBalance(ctx context.Context, user User) (int64, error) {
	ctx, _ = inslogger.WithTraceField(ctx, utils.RandTraceID())

	callParams := map[string]interface{}{
		"reference": user.Reference.String(),
	}
	callMethodReply, _, err := h.s.BasicAPICall(ctx, "member.getBalance", callParams, user.Reference, &user)
	if err != nil {
		return 0, errors.Wrap(err, "failed to call member.getBalance")
	}

	var result map[string]interface{}
	if cm, ok := callMethodReply.(*reply.CallMethod); !ok {
		return 0, errors.Wrapf(err, "unexpected type of return value %T", callMethodReply)
	} else if err := insolar.Deserialize(cm.Result, &result); err != nil {
		return 0, errors.Wrap(err, "failed to deserialize result")
	}

	r0, ok := result["Returns"]
	if ok && r0 != nil {
		if r1, ok := r0.([]interface{}); !ok {
			return 0, errors.Errorf("bad response: bad type of 'Returns' [%#v]", r0)
		} else if len(r1) != 2 {
			return 0, errors.Errorf("bad response: bad length of 'Returns' [%#v]", r0)
		} else if r2, ok := r1[0].(map[string]interface{}); !ok {
			return 0, errors.Errorf("bad response: bad type of first value [%#v]", r1)
		} else if r3, ok := r2["balance"]; !ok {
			return 0, errors.Errorf("bad response: absent balance field [%#v]", r2)
		} else if balanceRaw, ok := r3.(string); !ok {
			return 0, errors.Errorf("bad response: balance field expected to be a string [%#v]", r3)
		} else if balance, err := strconv.ParseInt(balanceRaw, 10, 0); err != nil {
			return 0, errors.Wrapf(err, "failed to parse balance [%#v]", balanceRaw)
		} else {
			return balance, nil
		}
	}

	r0, ok = result["Error"]
	if ok && r0 != nil {
		return 0, errors.Errorf("%T: %#v", r0, r0)
	}

	panic("unreachable")
}

func (h *ServerHelper) waitBalance(ctx context.Context, user User, balance int64) error {
	doneWaiting := false
	for i := 0; i < 100; i++ {
		balance, err := h.getBalance(ctx, user)
		if err != nil {
			return err
		}
		if balance == balance {
			doneWaiting = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !doneWaiting {
		return errors.New("failed to wait until balance match")
	}
	return nil
}

var (
	GenesisDirectory string
	FeeWalletUser    *User
)

func TestMain(m *testing.M) {
	flag.Parse()

	cleanup, directoryWithGenesis, err := mimic.GenerateBootstrap(true)
	if err != nil {
		panic("[ ERROR ] Failed to generate bootstrap files: " + err.Error())

	}
	defer cleanup()
	GenesisDirectory = directoryWithGenesis

	FeeWalletUser, err = loadMemberKeys(path.Join(directoryWithGenesis, "launchnet/configs/fee_member_keys.json"))
	if err != nil {
		panic("[ ERROR ] Failed to load Fee Member key: " + err.Error())
	}
	FeeWalletUser.Reference = genesisrefs.ContractFeeMember

	os.Exit(m.Run())
}
