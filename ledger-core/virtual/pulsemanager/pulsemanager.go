// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulsemanager

import (
	"context"
	"sync"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/appctl"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/insolar/nodestorage"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ appctl.Manager = &PulseManager{}

type PulseManager struct {
	NodeNet       network.NodeNetwork  `inject:""` //nolint:staticcheck
	NodeSetter    nodestorage.Modifier `inject:""`
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
	newPulse := ConvertForLegacy(pulseChange)
	ctx := context.Background()

	if err := m.setLegacy(ctx, newPulse); err != nil {
		return err
	}
	return m.setNewPulse(ctx, pulseChange)
}

func ConvertForLegacy(pc appctl.PulseChange) (psp pulsestor.Pulse) {
	pd := pc.Data

	copy(psp.Entropy[:32], pd.PulseEntropy[:])
	copy(psp.Entropy[32:], pd.PulseEntropy[:])

	psp.PulseNumber = pd.PulseNumber
	ok := false

	if psp.PrevPulseNumber, ok = pd.PulseNumber.TryPrev(pd.PrevPulseDelta); !ok {
		psp.PrevPulseNumber = pd.PulseNumber
	}

	if psp.NextPulseNumber, ok = pd.PulseNumber.TryNext(pd.NextPulseDelta); !ok {
		psp.NextPulseNumber = pd.PulseNumber
	}
	psp.EpochPulseNumber = pd.PulseEpoch
	psp.PulseTimestamp = int64(pd.Timestamp) * int64(time.Second)
	copy(psp.OriginID[:], pc.PulseOrigin)

	return psp
}

func (m *PulseManager) setNewPulse(ctx context.Context, pulseChange appctl.PulseChange) error {
	// pulseChange := appctl.PulseChange{
	// 	PulseSeq:  0,
	// 	Pulse:     pulse.NewOnePulseRange(pulse.Data{
	// 		PulseNumber: newPulse.PulseNumber,
	// 		DataExt:     pulse.DataExt{
	// 			PulseEpoch:     newPulse.EpochPulseNumber,
	// 			NextPulseDelta: uint16(newPulse.NextPulseNumber - newPulse.PulseNumber),
	// 			PrevPulseDelta: uint16(newPulse.PulseNumber - newPulse.PrevPulseNumber),
	// 			Timestamp:      uint32(newPulse.PulseTimestamp / int64(time.Second)),
	// 			PulseEntropy:   longbits.NewBits256FromBytes(newPulse.Entropy[:32]),
	// 		},
	// 	}),
	// 	StartedAt: time.Now(),
	// 	Census:    nil,
	// }


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

// Set set's new pulse in the old way.
func (m *PulseManager) setLegacy(ctx context.Context, newPulse pulsestor.Pulse) error {
	m.setLock.Lock()
	defer m.setLock.Unlock()
	if m.stopped {
		return errors.New("can't call Set method on Manager after stop")
	}

	storagePulse, err := m.PulseAccessor.Latest(ctx)
	switch {
	case err == pulsestor.ErrNotFound:
		storagePulse = appctl.PulseChange{}
	case err != nil:
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
		toSet := make([]rms.Node, 0, len(fromNetwork))
		for _, n := range fromNetwork {
			toSet = append(toSet, rms.Node{ID: rms.NewReference(n.ID()), Role: n.Role()})
		}
		err := m.NodeSetter.Set(newPulse.PulseNumber, toSet)
		if err != nil {
			panic(throw.W(err, "call of SetActiveNodes failed", nil))
		}
	}
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
