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
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/adapters"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ chorus.Conductor = &PulseManager{}
var _ adapters.NodeStater = &PulseManager{}

type PulseManager struct {
	NodeNet       network.NodeNetwork `inject:""` //nolint:staticcheck
	PulseAccessor beat.Accessor       `inject:""`
	PulseAppender beat.Appender       `inject:""`
	dispatchers   []beat.Dispatcher

	// mutex locks Set method call.
	mutex sync.RWMutex
	// saves PM stopping mode
	stopped bool
	ackFn   func(ack bool)
}

func NewPulseManager() *PulseManager {
	return &PulseManager{}
}

func (m *PulseManager) AddDispatcher(d ...beat.Dispatcher) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.stopped {
		panic(throw.IllegalState())
	}

	m.dispatchers = append(m.dispatchers, d...)
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

	ctx := context.Background()
	return m._commit(ctx, pulseChange)
}

func (m *PulseManager) CommitFirstPulseChange(pulseChange beat.Beat) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if ackFn := m.ackFn; ackFn != nil {
		panic(throw.IllegalState())
	}
	ctx := context.Background()
	return m._commit(ctx, pulseChange)
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

func (m *PulseManager) _commit(ctx context.Context, pulseChange beat.Beat) error {
	if err := m.PulseAppender.EnsureLatest(ctx, pulseChange); err != nil {
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
