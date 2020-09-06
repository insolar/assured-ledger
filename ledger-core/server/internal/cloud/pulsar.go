// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package cloud

import (
	"context"
	"sync"
	"time"

	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/keystore"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/network/transport"
	"github.com/insolar/assured-ledger/ledger-core/pulsar"
	"github.com/insolar/assured-ledger/ledger-core/pulsar/entropygenerator"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

type PulsarWrapper struct {
	cm     *component.Manager
	server *pulsar.Pulsar
	cfg    configuration.PulsarConfiguration
	ticker *time.Ticker
}

func NewPulsarWrapper(distributor pulsar.PulseDistributor, cfg configuration.PulsarConfiguration) *PulsarWrapper {
	ctx := context.Background()

	keyStore, err := keystore.NewKeyStore(cfg.KeysPath)
	if err != nil {
		panic(err)
	}
	cryptographyScheme := platformpolicy.NewPlatformCryptographyScheme()
	cryptographyService := platformpolicy.NewCryptographyService()
	keyProcessor := platformpolicy.NewKeyProcessor()

	cm := component.NewManager(nil)
	cm.SetLogger(global.Logger())

	cm.Register(cryptographyScheme, keyStore, keyProcessor, transport.NewFactory(cfg.Pulsar.DistributionTransport))
	cm.Inject(cryptographyService, distributor)

	if err = cm.Init(ctx); err != nil {
		panic(err)
	}

	if err = cm.Start(ctx); err != nil {
		panic(err)
	}

	server := pulsar.NewPulsar(
		cfg.Pulsar,
		cryptographyService,
		cryptographyScheme,
		keyProcessor,
		distributor,
		&entropygenerator.StandardEntropyGenerator{},
	)

	return &PulsarWrapper{
		cm:     cm,
		cfg:    cfg,
		server: server,
	}
}

var onePulsar sync.Once

func (p *PulsarWrapper) Start(ctx context.Context) error {
	onePulsar.Do(func() {
		p.ticker = runPulsar(ctx, p.server, p.cfg.Pulsar)
	})
	return nil
}

func (p *PulsarWrapper) Stop(ctx context.Context) error {
	p.ticker.Stop()
	return p.cm.Stop(ctx)
}

func runPulsar(ctx context.Context, server *pulsar.Pulsar, cfg configuration.Pulsar) *time.Ticker {
	nextPulseNumber := pulse.OfNow()
	err := server.Send(ctx, nextPulseNumber)
	if err != nil {
		panic(err)
	}

	pulseTicker := time.NewTicker(time.Duration(cfg.PulseTime) * time.Millisecond)
	go func() {
		for range pulseTicker.C {
			err := server.Send(ctx, server.LastPN()+pulse.Number(cfg.NumberDelta))
			if err != nil {
				panic(err)
			}
		}
	}()

	return pulseTicker
}
