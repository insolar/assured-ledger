// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulsemanager

import (
	"context"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/appctl"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

var _ appctl.Manager = &PulseManager{}

type PulseManager struct {
	NodeNet       network.NodeNetwork  `inject:""` //nolint:staticcheck
	PulseAccessor appctl.Accessor      `inject:""`
	PulseAppender appctl.Appender      `inject:""`
	dispatchers   []appctl.Dispatcher

	// setLock locks Set method call.
	setLock sync.RWMutex
	// saves PM stopping mode
	stopped bool
}

// NewPulseManager creates Manager instance.
func NewPulseManager() *PulseManager {
	return &PulseManager{}
}

// AddDispatcher adds dispatchers to handling
// that could be done only when Set is not happening
func (m *PulseManager) AddDispatcher(d ...appctl.Dispatcher) {
	m.setLock.Lock()
	defer m.setLock.Unlock()

	m.dispatchers = append(m.dispatchers, d...)
}

type messageNewPulse struct {
	*log.Msg `txt:"received pulse"`
	OldPulse pulse.Number
	NewPulse pulse.Number
}

func (m *PulseManager) CommitPulseChange(pulseChange appctl.PulseChange) error {
	ctx := context.Background()
	return m.setNewPulse(ctx, pulseChange)
}

func (m *PulseManager) setNewPulse(ctx context.Context, pulseChange appctl.PulseChange) error {

	sink, setStateFn := appctl.NewNodeStateSink(make(chan appctl.NodeState, 1))
	for _, d := range m.dispatchers {
		d.PreparePulseChange(pulseChange, sink)
	}
	committed := false

	defer func() {
		setStateFn(committed)
	}()

	if err := m.PulseAppender.Append(ctx, pulseChange); err != nil {
		return errors.W(err, "call of AddPulse failed")
	}

	for _, d := range m.dispatchers {
		d.CommitPulseChange(pulseChange)
	}
	committed = true

	return nil
}

// Start starts pulse manager.
func (m *PulseManager) Start(context.Context) error {
	return nil
}

// Stop stops Manager.
func (m *PulseManager) Stop(context.Context) error {
	// There should not to be any Set call after Stop call
	m.setLock.Lock()
	defer m.setLock.Unlock()

	m.stopped = true
	return nil
}
