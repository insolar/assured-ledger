package pulsar

import (
	"context"
	"sync"
	"time"

	"go.opencensus.io/stats"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/pulsar/entropygenerator"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/rms"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
)

// Pulsar is a base struct for pulsar's node
// It contains all the stuff, which is needed for working of a pulsar
type Pulsar struct {
	Config       configuration.Pulsar
	PublicKeyRaw string

	EntropyGenerator entropygenerator.EntropyGenerator

	Certificate                mandates.Certificate
	CryptographyService        cryptography.Service
	PlatformCryptographyScheme cryptography.PlatformCryptographyScheme
	KeyProcessor               cryptography.KeyProcessor
	PulseDistributor           PulseDistributor

	lastPNMutex sync.RWMutex
	lastPN      pulse.Number
}

// NewPulsar creates a new pulse with using of custom GeneratedEntropy Generator
func NewPulsar(
	configuration configuration.Pulsar,
	cryptographyService cryptography.Service,
	scheme cryptography.PlatformCryptographyScheme,
	keyProcessor cryptography.KeyProcessor,
	pulseDistributor PulseDistributor,
	entropyGenerator entropygenerator.EntropyGenerator,
) *Pulsar {

	global.Info("[NewPulsar]")

	pulsar := &Pulsar{
		CryptographyService:        cryptographyService,
		PlatformCryptographyScheme: scheme,
		KeyProcessor:               keyProcessor,
		PulseDistributor:           pulseDistributor,
		Config:                     configuration,
		EntropyGenerator:           entropyGenerator,
	}

	pubKey, err := cryptographyService.GetPublicKey()
	if err != nil {
		global.Fatal(err)
	}
	pubKeyRaw, err := keyProcessor.ExportPublicKeyPEM(pubKey)
	if err != nil {
		global.Fatal(err)
	}
	pulsar.PublicKeyRaw = string(pubKeyRaw)

	return pulsar
}

func (p *Pulsar) Send(ctx context.Context, pulseNumber pulse.Number) error {
	logger := inslogger.FromContext(ctx)
	logger.Infof("before sending new pulseNumber: %v", pulseNumber)

	entropy, _, err := p.generateNewEntropyAndSign()
	if err != nil {
		logger.Error(err)
		return err
	}

	pulseForSending := PulsePacket{
		PulseNumber:      pulseNumber,
		Entropy:          entropy,
		NextPulseNumber:  pulseNumber + pulse.Number(p.Config.NumberDelta),
		PrevPulseNumber:  p.lastPN,
		EpochPulseNumber: pulseNumber.AsEpoch(),
		OriginID:         [16]byte{206, 41, 229, 190, 7, 240, 162, 155, 121, 245, 207, 56, 161, 67, 189, 0},
		PulseTimestamp:   time.Now().UnixNano(),
		Signs:            map[string]SenderConfirmation{},
	}

	payload := PulseSenderConfirmationPayload{SenderConfirmation: SenderConfirmation{
		ChosenPublicKey: p.PublicKeyRaw,
		Entropy:         entropy,
		PulseNumber:     pulseNumber,
	}}
	hasher := platformpolicy.NewPlatformCryptographyScheme().IntegrityHasher()
	hash, err := payload.Hash(hasher)
	if err != nil {
		return err
	}
	signature, err := p.CryptographyService.Sign(hash)
	if err != nil {
		return err
	}

	pulseForSending.Signs[p.PublicKeyRaw] = SenderConfirmation{
		ChosenPublicKey: p.PublicKeyRaw,
		Signature:       signature.Bytes(),
		Entropy:         entropy,
		PulseNumber:     pulseNumber,
	}

	logger.Debug("Start a process of sending pulse")
	go func() {
		logger.Debug("Before sending to network")
		p.PulseDistributor.Distribute(ctx, pulseForSending)
	}()

	p.lastPNMutex.Lock()
	p.lastPN = pulseNumber
	p.lastPNMutex.Unlock()
	logger.Infof("set latest pulse: %v", pulseForSending.PulseNumber)

	stats.Record(ctx, statPulseGenerated.M(1), statCurrentPulse.M(int64(pulseNumber.AsUint32())))
	return nil
}

func (p *Pulsar) LastPN() pulse.Number {
	p.lastPNMutex.RLock()
	defer p.lastPNMutex.RUnlock()

	return p.lastPN
}

func (p *Pulsar) generateNewEntropyAndSign() (rms.Entropy, []byte, error) {
	e := p.EntropyGenerator.GenerateEntropy()

	sign, err := p.CryptographyService.Sign(e[:])
	if err != nil {
		return rms.Entropy{}, nil, err
	}

	return e, sign.Bytes(), nil
}

// PulseSenderConfirmationPayload is a struct with info about pulse's confirmations
type PulseSenderConfirmationPayload struct {
	SenderConfirmation
}

// Hash calculates hash of payload
func (ps *PulseSenderConfirmationPayload) Hash(hashProvider cryptography.Hasher) ([]byte, error) {
	_, err := hashProvider.Write(ps.PulseNumber.Bytes())
	if err != nil {
		return nil, err
	}
	_, err = hashProvider.Write([]byte(ps.ChosenPublicKey))
	if err != nil {
		return nil, err
	}
	_, err = hashProvider.Write(ps.Entropy[:])
	if err != nil {
		return nil, err
	}
	return hashProvider.Sum(nil), nil
}

/*

if currentPulsar.isStandalone() {
currentPulsar.SetCurrentSlotEntropy(currentPulsar.GetGeneratedEntropy())
currentPulsar.CurrentSlotPulseSender = currentPulsar.PublicKeyRaw

payload := PulseSenderConfirmationPayload{insolar.SenderConfirmation{
ChosenPublicKey: currentPulsar.CurrentSlotPulseSender,
Entropy:         *currentPulsar.GetCurrentSlotEntropy(),
Number:     currentPulsar.ProcessingPulseNumber,
}}
hashProvider := currentPulsar.PlatformCryptographyScheme.IntegrityHasher()
hash, err := payload.Hash(hashProvider)
if err != nil {
currentPulsar.StateSwitcher.SwitchToState(ctx, Failed, err)
return
}
signature, err := currentPulsar.Service.Sign(hash)
if err != nil {
currentPulsar.StateSwitcher.SwitchToState(ctx, Failed, err)
return
}

currentPulsar.currentSlotSenderConfirmationsLock.Lock()
currentPulsar.CurrentSlotSenderConfirmations[currentPulsar.PublicKeyRaw] = insolar.SenderConfirmation{
ChosenPublicKey: currentPulsar.CurrentSlotPulseSender,
Signature:       signature.Bytes(),
Entropy:         *currentPulsar.GetCurrentSlotEntropy(),
Number:     currentPulsar.ProcessingPulseNumber,
}
currentPulsar.currentSlotSenderConfirmationsLock.Unlock()

currentPulsar.StateSwitcher.SwitchToState(ctx, SendingPulse, nil)

return
}*/
