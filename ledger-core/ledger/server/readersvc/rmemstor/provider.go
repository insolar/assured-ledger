// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rmemstor

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/ledger/server/readersvc/readbundle"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ readbundle.Provider = &Provider{}

type Provider struct {
	mutex sync.RWMutex
	m map[pulse.Number]storageCabinet
}

func (p *Provider) FindCabinet(pn pulse.Number) readbundle.ReadCabinet {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	if cab, ok := p.m[pn]; ok {
		return cab
	}
	return nil
}

func (p *Provider) AddCabinet(pn pulse.Number, rd readbundle.Reader) error {
	switch {
	case !pn.IsTimePulse():
		panic(throw.IllegalValue())
	case rd == nil:
		panic(throw.IllegalValue())
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.m == nil {
		p.m = map[pulse.Number]storageCabinet{}
	} else if _, ok := p.m[pn]; ok {
		return throw.E("duplicate cabinet", struct { PN pulse.Number }{ pn })
	}

	p.m[pn] = storageCabinet{ pn, rd}
	return nil
}

type storageCabinet struct {
	pn pulse.Number
	rd readbundle.Reader
}

func (v storageCabinet) PulseNumber() pulse.Number {
	return v.pn
}

func (v storageCabinet) Open() (readbundle.Reader, error) {
	return v.rd, nil
}

func (v storageCabinet) Close() {}

