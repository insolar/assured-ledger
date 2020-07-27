// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulsemanager

import (
	"context"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/appctl/chorus"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ chorus.Conductor = &PulseManager{}

type PulseManager struct {
	NodeNet       network.NodeNetwork `inject:""` //nolint:staticcheck
	PulseAccessor beat.Accessor       `inject:""`
	PulseAppender beat.Appender       `inject:""`
	dispatchers   []beat.Dispatcher

	// setLock locks Set method call.
	setLock sync.RWMutex
	// saves PM stopping mode
	stopped bool
}

// NewPulseManager creates Conductor instance.
func NewPulseManager() *PulseManager {
	return &PulseManager{}
}

// AddDispatcher adds dispatchers to handling
// that could be done only when Set is not happening
func (m *PulseManager) AddDispatcher(d ...beat.Dispatcher) {
	m.setLock.Lock()
	defer m.setLock.Unlock()

	m.dispatchers = append(m.dispatchers, d...)
}

func (m *PulseManager) CommitPulseChange(pulseChange beat.Beat) error {
	ctx := context.Background()
	return m.setNewPulse(ctx, pulseChange)
}

func (m *PulseManager) CommitFirstPulseChange(pulseChange beat.Beat) error {
	ctx := context.Background()
	return m.setNewPulse(ctx, pulseChange)
}

func (m *PulseManager) setNewPulse(ctx context.Context, pulseChange beat.Beat) error {
	if err := m.PulseAppender.EnsureLatest(ctx, pulseChange); err != nil {
		return throw.W(err, "call of Ensure pulseChange failed")
	}

	sink, setStateFn := beat.NewAck(make(chan beat.AckData, 1))
	for _, d := range m.dispatchers {
		d.PrepareBeat(pulseChange, sink)
	}
	committed := false

	defer func() {
		setStateFn(committed)
	}()

	for _, d := range m.dispatchers {
		d.CommitBeat(pulseChange)
	}
	committed = true

	return nil
}

// Start starts pulse manager.
func (m *PulseManager) Start(context.Context) error {
	return nil
}

// Stop stops Conductor.
func (m *PulseManager) Stop(context.Context) error {
	// There should not to be any Set call after Stop call
	m.setLock.Lock()
	defer m.setLock.Unlock()

	m.stopped = true
	return nil
}
