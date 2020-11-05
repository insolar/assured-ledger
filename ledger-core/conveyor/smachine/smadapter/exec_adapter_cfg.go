// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smadapter

import (
	"context"
	"math"
	"runtime"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/managed"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type Config struct {
	MaxBufferCapacity int
	ExpectedParallelReaders int
	ProvideCancel bool
}

func NewExecutor(ctx context.Context, cfg Config, runArg interface{}) (exec smachine.AdapterExecutor, startFn func()) {
	parallelWorkers := uint16(0)
	chanLimit := cfg.ExpectedParallelReaders
	switch {
	case chanLimit >= math.MaxUint8:
		chanLimit = math.MaxUint8
		parallelWorkers = math.MaxUint8

	case chanLimit > 0:
		parallelWorkers = uint16(chanLimit)

	case chanLimit < 0:
		chanLimit = runtime.NumCPU() + 1
		// parallelWorkers = 0

	default:
		chanLimit = runtime.NumCPU()
		if chanLimit > 4 {
			chanLimit = 4
		}
		parallelWorkers = uint16(chanLimit)
	}

	var ch chan smachine.AdapterCall

	switch output := make(chan smachine.AdapterCall, chanLimit<<1); {
	case cfg.MaxBufferCapacity == 0:
		exec, ch = WrapCallChannelNoBuffer(cfg.ProvideCancel, output), output
	case cfg.MaxBufferCapacity < 0: // unlimited buffer
		exec, ch = WrapCallChannelNoLimit(ctx, cfg.ProvideCancel, output), output
	default:
		exec, ch = WrapCallChannel(ctx, cfg.MaxBufferCapacity, cfg.ProvideCancel, output), output
	}

	return exec, func() {
		smachine.StartChannelWorkerParallelCalls(ctx, parallelWorkers, ch, runArg)
	}
}

func StartExecutorFor(ctx context.Context, cfg Config, runArg interface{}) smachine.AdapterExecutor {
	exec, startFn := NewExecutor(ctx, cfg, runArg)
	startFn()
	return exec
}

func NewComponent(ctx context.Context, cfg Config, runArg interface{}, initFn func(managed.Holder)) (smachine.AdapterExecutor, managed.Component) {
	exec, ac := NewComponentExt(ctx, runArg, initFn, cfg)
	return exec[0], ac
}

func NewComponentExt(ctx context.Context, runArg interface{}, initFn func(managed.Holder), cfg ...Config) ([]smachine.AdapterExecutor, managed.Component) {
	if len(cfg) == 0 {
		panic(throw.IllegalValue())
	}

	ctx, stopFn := context.WithCancel(ctx)

	exec := make([]smachine.AdapterExecutor, len(cfg))
	startFn := make([]func(), len(cfg))

	for i := range cfg {
		exec[i], startFn[i] = NewExecutor(ctx, cfg[i], runArg)
	}

	ac := adapterComponent{initFn, nil, stopFn, nil }

	if len(startFn) == 1 {
		ac.startFn = startFn[0]
	} else {
		ac.startFn = func() {
			for _, fn := range startFn {
				fn()
			}
		}
	}

	switch c := runArg.(type) {
	case managed.ComponentWithPulse:
		ac.c = c
		return exec, adapterComponentWithPulse{ac}
	case managed.Component:
		ac.c = c
	}
	return exec, ac
}

type adapterComponent struct {
	initFn func(managed.Holder)
	startFn, stopFn func()
	c managed.Component
}

func (v adapterComponent) Init(holder managed.Holder) {
	if v.c != nil {
		v.c.Init(holder)
	}
	if v.initFn != nil {
		v.initFn(holder)
		v.initFn = nil
	}
}

func (v adapterComponent) Start(holder managed.Holder) {
	if v.c != nil {
		v.c.Start(holder)
	}
	v.startFn()
	v.startFn = nil
}

func (v adapterComponent) Stop(holder managed.Holder) {
	v.stopFn()
	if v.c != nil {
		v.c.Stop(holder)
	}
}

var _ managed.ComponentWithPulse = adapterComponentWithPulse{}
type adapterComponentWithPulse struct {
	adapterComponent
}

func (v adapterComponentWithPulse) PulseMigration(holder managed.Holder, p pulse.Range) {
	v.c.(managed.ComponentWithPulse).PulseMigration(holder, p)
}
