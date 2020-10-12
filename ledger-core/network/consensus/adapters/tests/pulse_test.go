// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package tests

import (
	"bytes"
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/crypto/legacyadapter"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/serialization"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

const (
	initialPulse = 100000
)

var digester = func() cryptkit.DataDigester {
	scheme := platformpolicy.NewPlatformCryptographyScheme()
	digester := legacyadapter.NewSha3Digester512(scheme)
	return digester
}()

var signer = func() cryptkit.DigestSigner {
	processor := platformpolicy.NewKeyProcessor()
	key, _ := processor.GeneratePrivateKey()
	scheme := platformpolicy.NewPlatformCryptographyScheme()
	signer := legacyadapter.NewECDSADigestSignerFromSK(key, scheme)
	return signer
}()

type Pulsar struct {
	pulseDelta  uint16
	pulseNumber pulse.Number
	// transports  []transport.DatagramTransport
	addresses []string

	mu *sync.Mutex
}

func NewPulsar(pulseDelta uint16, addresses []string, transports []transport.DatagramTransport) Pulsar {
	return Pulsar{
		pulseDelta:  pulseDelta,
		pulseNumber: initialPulse,
		addresses:   addresses,
		// transports:  transports,
		mu: &sync.Mutex{},
	}
}

func (p *Pulsar) Pulse(ctx context.Context, attempts int) {
	p.mu.Lock()
	defer time.AfterFunc(time.Duration(p.pulseDelta)*time.Second, func() {
		p.mu.Unlock()
	})

	prevDelta := p.pulseDelta
	if p.pulseNumber == initialPulse {
		prevDelta = 0
	}

	data := pulse.NewPulsarData(p.pulseNumber, p.pulseDelta, prevDelta, randBits256())
	p.pulseNumber += pulse.Number(p.pulseDelta)
	pp := serialization.BuildPulsarPacket(ctx, data)

	go func() {
		for i := 0; i < attempts; i++ {
			nodeID := rand.Intn(len(p.addresses))
			address := p.addresses[nodeID]
			transport := p.transports[nodeID]
			go func() {
				buffer := &bytes.Buffer{}
				_, err := pp.SerializeTo(ctx, buffer, digester, signer)
				if err != nil {
					panic(errors.W(err, "Failed to serialize packet"))
				}

				err = transport.SendDatagram(ctx, address, buffer.Bytes())
				if err != nil {
					panic(errors.W(err, "[SendDatagram] Failed to write data"))
				}

			}()
		}
	}()
}

func randBits256() longbits.Bits256 {
	v := longbits.Bits256{}
	_, _ = rand.Read(v[:])
	return v
}
