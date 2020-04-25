// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package executor

import (
	"context"
	"sync"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/flow/dispatcher"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/instracer"
	"github.com/insolar/assured-ledger/ledger-core/v2/network"
)

// PulseManager implements insolar.PulseManager.
type PulseManager struct {
	setLock sync.RWMutex

	nodeNet          network.NodeNetwork
	dispatchers      []dispatcher.Dispatcher
	nodeSetter       node.Modifier
	pulseAccessor    pulse.Accessor
	pulseAppender    pulse.Appender
	jetReleaser      JetReleaser
	writeManager     WriteManager
	hotStatusChecker HotDataStatusChecker
	registry         MetricsRegistry
}

// NewPulseManager creates PulseManager instance.
func NewPulseManager(
	nodeNet network.NodeNetwork,
	dispatchers []dispatcher.Dispatcher,
	nodeSetter node.Modifier,
	pulseAccessor pulse.Accessor,
	pulseAppender pulse.Appender,
	jetReleaser JetReleaser,
	writeManager WriteManager,
	hotStatusChecker HotDataStatusChecker,
	registry MetricsRegistry,
) *PulseManager {
	pm := &PulseManager{
		nodeNet:          nodeNet,
		dispatchers:      dispatchers,
		nodeSetter:       nodeSetter,
		pulseAccessor:    pulseAccessor,
		pulseAppender:    pulseAppender,
		jetReleaser:      jetReleaser,
		writeManager:     writeManager,
		hotStatusChecker: hotStatusChecker,
		registry:         registry,
	}
	return pm
}

// Set set's new pulse and closes current jet drop.
func (m *PulseManager) Set(ctx context.Context, newPulse insolar.Pulse) error {
	logger := inslogger.FromContext(ctx)
	logger.WithFields(map[string]interface{}{
		"new_pulse": newPulse.PulseNumber,
	}).Info("trying to set new pulse")

	m.setLock.Lock()
	defer m.setLock.Unlock()

	logger.Debug("behind set lock")
	ctx, span := instracer.StartSpan(ctx, "PulseManager.Set")
	span.SetTag("pulse.PulseNumber", int64(newPulse.PulseNumber))
	defer span.Finish()

	// Dealing with node lists.
	logger.Debug("dealing with node lists.")
	{
		fromNetwork := m.nodeNet.GetAccessor(newPulse.PulseNumber).GetWorkingNodes()
		if len(fromNetwork) == 0 {
			logger.Errorf("received zero nodes for pulse %d", newPulse.PulseNumber)
			return nil
		}
		toSet := make([]insolar.Node, 0, len(fromNetwork))
		for _, n := range fromNetwork {
			toSet = append(toSet, insolar.Node{ID: n.ID(), Role: n.Role()})
		}
		err := m.nodeSetter.Set(newPulse.PulseNumber, toSet)
		if err != nil {
			logger.Panic(errors.Wrap(err, "call of SetActiveNodes failed"))
		}
	}

	endedPulse, err := m.pulseAccessor.Latest(ctx)
	if err != nil {
		logger.Panic(errors.Wrap(err, "failed to fetch ended pulse"))
	}

	// Changing pulse.
	logger.Debug("before changing pulse")
	{
		logger.Debug("before dispatcher closePulse")
		for _, d := range m.dispatchers {
			d.ClosePulse(ctx, newPulse)
		}

		logger.WithFields(map[string]interface{}{
			"endedPulse": endedPulse.PulseNumber,
		}).Debugf("before jetReleaser.CloseAllUntil")
		m.jetReleaser.CloseAllUntil(ctx, endedPulse.PulseNumber)

		logger.WithFields(map[string]interface{}{
			"endedPulse": endedPulse.PulseNumber,
		}).Debugf("before writeManager.CloseAndWait")
		err = m.writeManager.CloseAndWait(ctx, endedPulse.PulseNumber)
		if err != nil {
			logger.Panic(errors.Wrap(err, "can't close pulse for writing"))
		}

		logger.WithField("newPulse.PulseNumber", newPulse.PulseNumber).Debug("before writeManager.Open")
		err = m.writeManager.Open(ctx, newPulse.PulseNumber)
		if err != nil {
			logger.Panic(errors.Wrap(err, "failed to open pulse for writing"))
		}

		logger.WithField("newPulse.PulseNumber", newPulse.PulseNumber).Debug("before pulseAppender.Append")
		if err := m.pulseAppender.Append(ctx, newPulse); err != nil {
			logger.Panic(errors.Wrap(err, "failed to add pulse"))
		}

		logger.WithField("newPulse", newPulse.PulseNumber).Debugf("before dispatcher.BeginPulse", newPulse)
		for _, d := range m.dispatchers {
			d.BeginPulse(ctx, newPulse)
		}
	}

	// set metrics and reset all counters
	m.registry.UpdateMetrics(ctx)

	logger.Info("new pulse is set")
	return nil
}
