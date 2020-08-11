// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package consensus

import (
	"context"
	"fmt"
	"reflect"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/crypto/legacyadapter"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/adapters"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	transport2 "github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/transport"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/censusimpl"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/core"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/core/coreapi"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/serialization"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network/transport"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

type packetProcessorSetter interface {
	SetPacketProcessor(adapters.PacketProcessor)
	SetPacketParserFactory(factory adapters.PacketParserFactory)
}

type Mode uint

const (
	ReadyNetwork = Mode(iota)
	Joiner
)

func New(ctx context.Context, dep Dep) Installer {
	ctx = adapters.ConsensusContext(ctx)
	dep.verify()

	constructor := newConstructor(ctx, &dep)
	constructor.verify()

	return newInstaller(constructor, &dep)
}

func verify(s interface{}) {
	cdValue := reflect.Indirect(reflect.ValueOf(s))
	cdType := cdValue.Type()

	for i := 0; i < cdValue.NumField(); i++ {
		fieldMeta := cdValue.Field(i)

		if (fieldMeta.Kind() == reflect.Interface || fieldMeta.Kind() == reflect.Ptr) && fieldMeta.IsNil() {
			panic(fmt.Sprintf("%s field %s is nil", cdType.Name(), cdType.Field(i).Name))
		}
	}
}

type Dep struct {
	KeyProcessor          cryptography.KeyProcessor
	CertificateManager    nodeinfo.CertificateManager
	KeyStore              cryptography.KeyStore
	TransportCryptography transport2.CryptographyAssistant

	NodeKeeper        beat.NodeKeeper
	DatagramTransport transport.DatagramTransport

	StateGetter         adapters.NodeStater
	PulseChanger        adapters.BeatChanger
	StateUpdater        adapters.StateUpdater
	EphemeralController adapters.EphemeralController

	LocalNodeProfile    profiles.StaticProfile
}

func (cd *Dep) verify() {
	verify(cd)
}

type constructor struct {
	consensusConfiguration census.ConsensusConfiguration
	mandateRegistry        census.MandateRegistry
	misbehaviorRegistry    census.MisbehaviorRegistry
	offlinePopulation      census.OfflinePopulation
	versionedRegistries    census.VersionedRegistries
	nodeProfileFactory     profiles.Factory
	localNodeConfiguration api.LocalNodeConfiguration
	roundStrategyFactory   core.RoundStrategyFactory

	packetBuilder                transport2.PacketBuilder
	packetSender                 transport2.PacketSender
	transportFactory             transport2.Factory
	transportCryptographyFactory transport2.CryptographyAssistant
}

func newConstructor(ctx context.Context, dep *Dep) *constructor {
	c := &constructor{}

	c.consensusConfiguration = adapters.NewConsensusConfiguration()
	c.mandateRegistry = adapters.NewMandateRegistry(
		cryptkit.NewDigest(
			longbits.NewBits512FromBytes(
				make([]byte, 64),
			),
			legacyadapter.SHA3Digest512,
		).AsDigestHolder(),
		c.consensusConfiguration,
	)
	c.misbehaviorRegistry = adapters.NewMisbehaviorRegistry()
	c.offlinePopulation = adapters.NewOfflinePopulation(dep.NodeKeeper) // TODO should use mandate storage

	c.versionedRegistries = adapters.NewVersionedRegistries(
		c.mandateRegistry,
		c.misbehaviorRegistry,
		c.offlinePopulation,
	)
	c.nodeProfileFactory = adapters.NewNodeProfileFactory(dep.KeyProcessor)
	c.localNodeConfiguration = adapters.NewLocalNodeConfiguration(
		ctx,
		dep.KeyStore,
	)
	c.roundStrategyFactory = adapters.NewRoundStrategyFactory()
	c.transportCryptographyFactory = dep.TransportCryptography
	c.packetBuilder = serialization.NewPacketBuilder(
		c.transportCryptographyFactory,
		c.localNodeConfiguration,
	)
	c.packetSender = adapters.NewPacketSender(dep.DatagramTransport)
	c.transportFactory = adapters.NewTransportFactory(
		c.transportCryptographyFactory,
		c.packetBuilder,
		c.packetSender,
	)

	return c
}

func (c *constructor) verify() {
	verify(c)
}

type Installer struct {
	dep       *Dep
	consensus *constructor
}

func newInstaller(constructor *constructor, dep *Dep) Installer {
	return Installer{
		dep:       dep,
		consensus: constructor,
	}
}

func (c Installer) ControllerFor(mode Mode, setters ...packetProcessorSetter) Controller {
	controlFeederInterceptor := adapters.InterceptConsensusControl(
		adapters.NewConsensusControlFeeder(),
	)

	cert := c.dep.CertificateManager.GetCertificate()
	isDiscovery := network.OriginIsDiscovery(cert)

	var candidateQueueSize int
	if isDiscovery {
		candidateQueueSize = 1
	}
	candidateFeeder := coreapi.NewSequentialCandidateFeeder(candidateQueueSize)

	var pop census.OnlinePopulation
	if na := c.dep.NodeKeeper.FindAnyLatestNodeSnapshot(); na != nil {
		pop = na.GetPopulation()
	}

	var ephemeralFeeder api.EphemeralControlFeeder
	if c.dep.EphemeralController.EphemeralMode(pop) {
		ephemeralFeeder = adapters.NewEphemeralControlFeeder(c.dep.EphemeralController)
	}

	upstreamController := adapters.NewUpstreamPulseController(
		c.dep.StateGetter,
		c.dep.PulseChanger,
		c.dep.StateUpdater,
	)

	consensusChronicles := c.createConsensusChronicles(mode, pop)
	consensusController := c.createConsensusController(
		consensusChronicles,
		controlFeederInterceptor.Feeder(),
		candidateFeeder,
		ephemeralFeeder,
		upstreamController,
	)
	packetParserFactory := c.createPacketParserFactory()

	c.bind(setters, consensusController, packetParserFactory)

	consensusController.Prepare()

	return newController(controlFeederInterceptor, candidateFeeder, consensusController, upstreamController, consensusChronicles)
}

func (c *Installer) createCensus(mode Mode, pop census.OnlinePopulation) *censusimpl.PrimingCensusTemplate {

	var nodes []profiles.StaticProfile
	if pop != nil {
		nodes = nodeinfo.NewStaticProfileList(pop.GetProfiles())
	}

	localProfile := c.dep.LocalNodeProfile
	if len(nodes) == 0 {
		nodes = append(nodes, localProfile)
	}

	if mode == Joiner {
		return adapters.NewCensusForJoiner(localProfile,
			c.consensus.versionedRegistries, c.consensus.transportCryptographyFactory,
		)
	}

	return adapters.NewCensus(localProfile, nodes,
		c.consensus.versionedRegistries, c.consensus.transportCryptographyFactory,
	)
}

func (c *Installer) createConsensusChronicles(mode Mode, pop census.OnlinePopulation) censusimpl.LocalConsensusChronicles {
	consensusChronicles := adapters.NewChronicles(c.consensus.nodeProfileFactory)
	c.createCensus(mode, pop).SetAsActiveTo(consensusChronicles)
	return consensusChronicles
}

func (c *Installer) createConsensusController(
	consensusChronicles censusimpl.LocalConsensusChronicles,
	controlFeeder api.ConsensusControlFeeder,
	candidateFeeder api.CandidateControlFeeder,
	ephemeralFeeder api.EphemeralControlFeeder,
	upstreamController api.UpstreamController,
) api.ConsensusController {
	return gcpv2.NewConsensusMemberController(
		consensusChronicles,
		upstreamController,
		core.NewPhasedRoundControllerFactory(
			c.consensus.localNodeConfiguration,
			c.consensus.transportFactory,
			c.consensus.roundStrategyFactory,
		),
		candidateFeeder,
		controlFeeder,
		ephemeralFeeder,
	)
}

func (c *Installer) createPacketParserFactory() adapters.PacketParserFactory {
	return serialization.NewPacketParserFactory(
		c.consensus.transportCryptographyFactory.GetDigestFactory().CreateDataDigester(),
		c.consensus.transportCryptographyFactory.CreateNodeSigner(c.consensus.localNodeConfiguration.GetSecretKeyStore()).GetSigningMethod(),
		c.dep.KeyProcessor,
	)
}

func (c *Installer) bind(
	setters []packetProcessorSetter,
	packetProcessor adapters.PacketProcessor,
	packetParserFactory adapters.PacketParserFactory,
) {
	for _, setter := range setters {
		setter.SetPacketProcessor(packetProcessor)
		setter.SetPacketParserFactory(packetParserFactory)
	}
}
