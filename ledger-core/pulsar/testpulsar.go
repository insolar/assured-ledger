// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulsar

import (
	"context"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/pulsar/entropygenerator"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

type TestPulsar struct {
	distributor   node.PulseDistributor
	generator     entropygenerator.EntropyGenerator
	configuration configuration.Pulsar
}

func NewTestPulsar(
	configuration configuration.Pulsar,
	distributor node.PulseDistributor,
	generator entropygenerator.EntropyGenerator,
) *TestPulsar {
	return &TestPulsar{
		distributor:   distributor,
		generator:     generator,
		configuration: configuration,
	}
}

func (p *TestPulsar) SendPulse(ctx context.Context) error {
	timeNow := time.Now()
	pulseNumber := pulse.OfTime(timeNow)

	pls := pulsestor.Pulse{
		PulseNumber:      pulseNumber,
		Entropy:          p.generator.GenerateEntropy(),
		NextPulseNumber:  pulseNumber + pulse.Number(p.configuration.NumberDelta),
		PrevPulseNumber:  pulseNumber - pulse.Number(p.configuration.NumberDelta),
		EpochPulseNumber: pulseNumber.AsEpoch(),
		OriginID:         [16]byte{206, 41, 229, 190, 7, 240, 162, 155, 121, 245, 207, 56, 161, 67, 189, 0},
	}

	var err error
	pls.Signs, err = getPSC(pls)
	if err != nil {
		global.Errorf("[ distribute ]", err)
		return err
	}

	pls.PulseTimestamp = time.Now().UnixNano()

	p.distributor.Distribute(ctx, pls)

	return nil
}

func getPSC(pulse pulsestor.Pulse) (map[string]pulsestor.SenderConfirmation, error) {
	proc := platformpolicy.NewKeyProcessor()
	key, err := proc.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}
	pem, err := proc.ExportPublicKeyPEM(proc.ExtractPublicKey(key))
	if err != nil {
		return nil, err
	}
	result := make(map[string]pulsestor.SenderConfirmation)
	psc := pulsestor.SenderConfirmation{
		PulseNumber:     pulse.PulseNumber,
		ChosenPublicKey: string(pem),
		Entropy:         pulse.Entropy,
	}

	payload := PulseSenderConfirmationPayload{SenderConfirmation: psc}
	hasher := platformpolicy.NewPlatformCryptographyScheme().IntegrityHasher()
	hash, err := payload.Hash(hasher)
	if err != nil {
		return nil, err
	}
	service := platformpolicy.NewKeyBoundCryptographyService(key)
	sign, err := service.Sign(hash)
	if err != nil {
		return nil, err
	}

	psc.Signature = sign.Bytes()
	result[string(pem)] = psc

	return result, nil
}
