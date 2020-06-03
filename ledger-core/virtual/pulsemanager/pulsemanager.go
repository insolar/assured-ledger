// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulsemanager

import (
	"context"
	"sync"

	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/insolar/dispatcher"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/nodestorage"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

// Manager implements insolar.Manager.
type PulseManager struct {
	NodeNet       network.NodeNetwork  `inject:""` //nolint:staticcheck
	NodeSetter    nodestorage.Modifier `inject:""`
	PulseAccessor pulsestor.Accessor   `inject:""`
	PulseAppender pulsestor.Appender   `inject:""`
	dispatchers   []dispatcher.Dispatcher

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
func (m *PulseManager) AddDispatcher(d ...dispatcher.Dispatcher) {
	m.setLock.Lock()
	defer m.setLock.Unlock()

	m.dispatchers = append(m.dispatchers, d...)
}

type messageNewPulse struct {
	*log.Msg `txt:"received pulse"`
	OldPulse pulse.Number
	NewPulse pulse.Number
}

// Set set's new pulse.
func (m *PulseManager) Set(ctx context.Context, newPulse pulsestor.Pulse) error {
	m.setLock.Lock()
	defer m.setLock.Unlock()
	if m.stopped {
		return errors.New("can't call Set method on Manager after stop")
	}

	storagePulse, err := m.PulseAccessor.Latest(ctx)
	if err == pulsestor.ErrNotFound {
		storagePulse = *pulsestor.GenesisPulse
	} else if err != nil {
		return errors.W(err, "call of GetLatestPulseNumber failed")
	}

	logger := inslogger.FromContext(ctx)
	logger.Debug(messageNewPulse{OldPulse: storagePulse.PulseNumber, NewPulse: newPulse.PulseNumber})

	{ // Dealing with node lists.
		fromNetwork := m.NodeNet.GetAccessor(newPulse.PulseNumber).GetWorkingNodes()
		if len(fromNetwork) == 0 {
			logger.Errorf("received zero nodes for pulse %d", newPulse.PulseNumber)
			return nil
		}
		toSet := make([]node.Node, 0, len(fromNetwork))
		for _, n := range fromNetwork {
			toSet = append(toSet, node.Node{ID: n.ID(), Role: n.Role()})
		}
		err := m.NodeSetter.Set(newPulse.PulseNumber, toSet)
		if err != nil {
			panic(throw.W(err, "call of SetActiveNodes failed", nil))
		}
	}

	for _, d := range m.dispatchers {
		d.ClosePulse(ctx, storagePulse)
	}

	if err := m.PulseAppender.Append(ctx, newPulse); err != nil {
		return errors.W(err, "call of AddPulse failed")
	}

	for _, d := range m.dispatchers {
		d.BeginPulse(ctx, newPulse)
	}

	return nil
}

// Start starts pulse manager.
func (m *PulseManager) Start(ctx context.Context) error {
	return nil
}

// Stop stops Manager.
func (m *PulseManager) Stop(ctx context.Context) error {
	// There should not to be any Set call after Stop call
	m.setLock.Lock()
	defer m.setLock.Unlock()

	m.stopped = true
	return nil
}
