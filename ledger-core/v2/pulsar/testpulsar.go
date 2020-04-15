// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulsar

import (
	"context"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/configuration"
	"github.com/insolar/assured-ledger/ledger-core/v2/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/global"
	"github.com/insolar/assured-ledger/ledger-core/v2/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulsar/entropygenerator"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
)

type TestPulsar struct {
	distributor   insolar.PulseDistributor
	generator     entropygenerator.EntropyGenerator
	configuration configuration.Pulsar
}

func NewTestPulsar(
	configuration configuration.Pulsar,
	distributor insolar.PulseDistributor,
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

	pls := insolar.Pulse{
		PulseNumber:      pulseNumber,
		Entropy:          p.generator.GenerateEntropy(),
		NextPulseNumber:  pulseNumber + insolar.PulseNumber(p.configuration.NumberDelta),
		PrevPulseNumber:  pulseNumber - insolar.PulseNumber(p.configuration.NumberDelta),
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

func getPSC(pulse insolar.Pulse) (map[string]insolar.PulseSenderConfirmation, error) {
	proc := platformpolicy.NewKeyProcessor()
	key, err := proc.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}
	pem, err := proc.ExportPublicKeyPEM(proc.ExtractPublicKey(key))
	if err != nil {
		return nil, err
	}
	result := make(map[string]insolar.PulseSenderConfirmation)
	psc := insolar.PulseSenderConfirmation{
		PulseNumber:     pulse.PulseNumber,
		ChosenPublicKey: string(pem),
		Entropy:         pulse.Entropy,
	}

	payload := PulseSenderConfirmationPayload{PulseSenderConfirmation: psc}
	hasher := platformpolicy.NewPlatformCryptographyScheme().IntegrityHasher()
	hash, err := payload.Hash(hasher)
	if err != nil {
		return nil, err
	}
	service := cryptography.NewKeyBoundCryptographyService(key)
	sign, err := service.Sign(hash)
	if err != nil {
		return nil, err
	}

	psc.Signature = sign.Bytes()
	result[string(pem)] = psc

	return result, nil
}
