// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulsestor

import (
	"context"
	"errors"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
)

type StartPulse interface {
	SetStartPulse(context.Context, Pulse)
	PulseNumber() (pulse.Number, error)
}

type startPulse struct {
	sync.RWMutex
	pulse *Pulse
}

func NewStartPulse() StartPulse {
	return &startPulse{}
}

func (sp *startPulse) SetStartPulse(ctx context.Context, pulse Pulse) {
	sp.Lock()
	defer sp.Unlock()

	if sp.pulse == nil {
		sp.pulse = &pulse
	}
}

func (sp *startPulse) PulseNumber() (pulse.Number, error) {
	sp.RLock()
	defer sp.RUnlock()

	if sp.pulse == nil {
		return 0, errors.New("start pulse in nil")
	}
	return sp.pulse.PulseNumber, nil
}
