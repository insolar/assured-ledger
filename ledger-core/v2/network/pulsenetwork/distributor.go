// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulsenetwork

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"go.opencensus.io/stats"

	"github.com/insolar/assured-ledger/ledger-core/v2/configuration"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/instracer"
	"github.com/insolar/assured-ledger/ledger-core/v2/metrics"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/adapters"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/serialization"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/hostnetwork/future"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/hostnetwork/host"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/sequence"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/transport"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type distributor struct {
	Factory  transport.Factory                  `inject:""`
	Scheme   insolar.PlatformCryptographyScheme `inject:""`
	KeyStore insolar.KeyStore                   `inject:""`

	digester    cryptkit.DataDigester
	signer      cryptkit.DigestSigner
	transport   transport.DatagramTransport
	idGenerator sequence.Generator

	pulseRequestTimeout time.Duration

	publicAddress   string
	pulsarHost      *host.Host
	bootstrapHosts  []string
	futureManager   future.Manager
	responseHandler future.PacketHandler
}

type handlerThatPanics struct{}

func (ph handlerThatPanics) HandleDatagram(ctx context.Context, address string, buf []byte) {
	panic(throw.Impossible())
}

// NewDistributor creates a new distributor object of pulses
func NewDistributor(conf configuration.PulseDistributor) (insolar.PulseDistributor, error) {
	futureManager := future.NewManager()

	result := &distributor{
		idGenerator: sequence.NewGenerator(),

		pulseRequestTimeout: time.Duration(conf.PulseRequestTimeout) * time.Millisecond,

		bootstrapHosts:  conf.BootstrapHosts,
		futureManager:   futureManager,
		responseHandler: future.NewPacketHandler(futureManager),
	}

	return result, nil
}

func (d *distributor) Init(ctx context.Context) error {
	var err error
	d.transport, err = d.Factory.CreateDatagramTransport(handlerThatPanics{})
	if err != nil {
		return errors.Wrap(err, "Failed to create transport")
	}
	transportCryptographyFactory := adapters.NewTransportCryptographyFactory(d.Scheme)

	d.digester = transportCryptographyFactory.GetDigestFactory().CreateDataDigester()

	privateKey, err := d.KeyStore.GetPrivateKey("")
	if err != nil {
		return errors.Wrap(err, "failed to get private key")
	}
	ecdsaPrivateKey := privateKey.(*ecdsa.PrivateKey)

	d.signer = adapters.NewECDSADigestSigner(ecdsaPrivateKey, d.Scheme)

	return nil
}

func (d *distributor) Start(ctx context.Context) error {

	err := d.transport.Start(ctx)
	if err != nil {
		return err
	}
	d.publicAddress = d.transport.Address()

	pulsarHost, err := host.NewHost(d.publicAddress)
	if err != nil {
		return errors.Wrap(err, "[ NewDistributor ] failed to create pulsar host")
	}
	pulsarHost.NodeID = insolar.NewEmptyReference()

	d.pulsarHost = pulsarHost
	return nil
}

func (d *distributor) Stop(ctx context.Context) error {
	return d.transport.Stop(ctx)
}

// Distribute starts a fire-and-forget process of pulse distribution to bootstrap hosts
func (d *distributor) Distribute(ctx context.Context, pulse insolar.Pulse) {
	logger := inslogger.FromContext(ctx)
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("sendPulseToNetwork failed with panic: %v", r)
		}
	}()

	pulseCtx := inslogger.SetLogger(context.Background(), logger)

	traceID := strconv.FormatUint(uint64(pulse.PulseNumber), 10) + "_pulse"
	pulseCtx, logger = inslogger.WithTraceField(pulseCtx, traceID)

	pulseCtx, span := instracer.StartSpan(pulseCtx, "Pulsar.Distribute")
	span.LogFields(
		log.Int64("pulse.PulseNumber", int64(pulse.PulseNumber)),
	)
	defer span.Finish()

	wg := sync.WaitGroup{}
	wg.Add(len(d.bootstrapHosts))

	distributed := int32(0)
	for _, nodeAddr := range d.bootstrapHosts {
		go func(ctx context.Context, pulse insolar.Pulse, nodeAddr string) {
			defer wg.Done()

			err := d.sendPulseToHost(ctx, &pulse, nodeAddr)
			if err != nil {
				stats.Record(ctx, statSendPulseErrorsCount.M(1))
				logger.Warnf("Failed to send pulse %d to host: %s %s", pulse.PulseNumber, nodeAddr, err)
				return
			}

			atomic.AddInt32(&distributed, 1)
			logger.Infof("Successfully sent pulse %d to node %s", pulse.PulseNumber, nodeAddr)
		}(pulseCtx, pulse, nodeAddr)
	}
	wg.Wait()

	if distributed == 0 {
		logger.Warn("No bootstrap hosts to distribute")
	} else {
		logger.Infof("Pulse distributed to %d hosts", distributed)
	}

}

// func (d *distributor) generateID() types.RequestID {
// 	return types.RequestID(d.idGenerator.Generate())
// }

func (d *distributor) sendPulseToHost(ctx context.Context, p *insolar.Pulse, host string) error {
	logger := inslogger.FromContext(ctx)
	defer func() {
		if x := recover(); x != nil {
			logger.Errorf("sendPulseToHost failed with panic: %v", x)
		}
	}()

	ctx, span := instracer.StartSpan(ctx, "distributor.sendPulseToHosts")
	defer span.Finish()

	pulsePacket := serialization.BuildPulsarPacket(ctx, adapters.NewPulseData(*p))

	err := d.sendRequestToHost(ctx, pulsePacket, host)
	if err != nil {
		return err
	}
	return nil
}

func (d *distributor) sendRequestToHost(ctx context.Context, p *serialization.Packet, rcv string) error {
	inslogger.FromContext(ctx).Debugf("Send %s request to %s",
		p.Header.GetPacketType(), rcv)

	buffer := &bytes.Buffer{}
	n, err := p.SerializeTo(ctx, buffer, d.digester, d.signer)
	if err != nil {
		return errors.Wrap(err, "Failed to serialize packet")
	}

	err = d.transport.SendDatagram(ctx, rcv, buffer.Bytes())
	if err != nil {
		return errors.Wrap(err, "[SendDatagram] Failed to write data")
	}

	metrics.NetworkSentSize.Observe(float64(n))
	metrics.NetworkPacketSentTotal.WithLabelValues(p.Header.GetPacketType().String()).Inc()

	return nil
}
