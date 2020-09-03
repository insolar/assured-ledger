// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insapp

import (
	"context"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/appctl/chorus"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ chorus.Conductor = &PulseManager{}

type PulseManager struct {
	PulseAppender beat.Appender       `inject:""`

	mutex sync.RWMutex
	dispatchers   []beat.Dispatcher
	stopped bool
	ackFn   func(ack bool)
}

func NewPulseManager() *PulseManager {
	return &PulseManager{}
}

func (m *PulseManager) AddDispatcher(d beat.Dispatcher) {
	if d == nil {
		panic(throw.IllegalValue())
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.stopped {
		panic(throw.IllegalState())
	}

	m.dispatchers = append(m.dispatchers, d)
}

func (m *PulseManager) CommitPulseChange(pulseChange beat.Beat) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if ackFn := m.ackFn; ackFn != nil {
		m.ackFn = nil
		ackFn(true)
	// } else {
	// 	panic(throw.IllegalState())
	}

	return m._commit(pulseChange)
}

func (m *PulseManager) CommitFirstPulseChange(pulseChange beat.Beat) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if ackFn := m.ackFn; ackFn != nil {
		panic(throw.IllegalState())
	}
	return m._commit(pulseChange)
}

func (m *PulseManager) RequestNodeState(stateFunc chorus.NodeStateFunc) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	sink, ackFn := beat.NewAck(func(data beat.AckData) {
		stateFunc(data.UpstreamState)
	})

	for _, d := range m.dispatchers {
		d.PrepareBeat(sink)
	}

	// if !sink.IsAcquired() {
	// 	panic(throw.IllegalState())
	// }

	m.ackFn = ackFn
}

func (m *PulseManager) CancelNodeState() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if ackFn := m.ackFn; ackFn != nil {
		m.ackFn = nil
		ackFn(false)
		return
	}
	panic(throw.IllegalState())
}

func (m *PulseManager) _commit(pulseChange beat.Beat) error {
	if err := m.PulseAppender.EnsureLatestTimeBeat(pulseChange); err != nil {
		return throw.W(err, "call of Ensure pulseChange failed")
	}

	for _, d := range m.dispatchers {
		d.CommitBeat(pulseChange)
	}

	return nil
}

func (m *PulseManager) Start(context.Context) error {
	return nil
}

func (m *PulseManager) Stop(context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.stopped = true
	return nil
}
