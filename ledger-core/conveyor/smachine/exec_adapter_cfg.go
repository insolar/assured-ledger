// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"context"
	"math"
	"runtime"
)

type AdapterExecutorConfig struct {
	MaxBufferCapacity int
	ExpectedParallelReaders int
	ProvideCancel bool
}

func NewAdapterExecutor(ctx context.Context, cfg AdapterExecutorConfig, runArg interface{}) (exec AdapterExecutor, startFn func()) {
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

	var ch chan AdapterCall

	switch output := make(chan AdapterCall, chanLimit<<1); {
	case cfg.MaxBufferCapacity == 0:
		exec, ch = WrapCallChannelNoBuffer(cfg.ProvideCancel, output), output
	case cfg.MaxBufferCapacity < 0: //unlimited buffer
		exec, ch = WrapCallChannelNoLimit(ctx, cfg.ProvideCancel, output), output
	default:
		exec, ch = WrapCallChannel(ctx, cfg.MaxBufferCapacity, cfg.ProvideCancel, output), output
	}

	return exec, func() {
		StartChannelWorkerParallelCalls(ctx, parallelWorkers, ch, runArg)
	}
}

func StartAdapterExecutor(ctx context.Context, cfg AdapterExecutorConfig, runArg interface{}) AdapterExecutor {
	exec, startFn := NewAdapterExecutor(ctx, cfg, runArg)
	startFn()
	return exec
}
