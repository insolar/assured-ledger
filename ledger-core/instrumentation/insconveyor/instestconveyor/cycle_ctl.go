// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package instestconveyor

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewCycleController() (*CycleController, conveyor.PulseConveyorCycleFunc) {
	ctl := &CycleController{}
	return &CycleController{}, ctl.onConveyorCycle
}

type CycleController struct {
	debugLock  sync.Mutex
	debugFlags atomickit.Uint32
	suspend    synckit.ClosableSignalChannel
	cycleFn    cycleFunc
}

type cycleFunc func(hasActive, isIdle bool)

func (s *CycleController) WaitIdleConveyor() {
	s.waitIdleConveyor(false)
}

func (s *CycleController) WaitActiveThenIdleConveyor() {
	s.waitIdleConveyor(true)
}

func (s *CycleController) waitIdleConveyor(checkActive bool) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	s.setWaitCallback(func(hadActive, isIdle bool) {
		if checkActive && !hadActive {
			return
		}
		if isIdle {
			s.setWaitCallback(nil)
			wg.Done()
		}
	})
	wg.Wait()
}

func (s *CycleController) SuspendConveyorNoWait() {
	s.debugLock.Lock()
	defer s.debugLock.Unlock()

	if s.suspend == nil {
		s.suspend = make(synckit.ClosableSignalChannel)
	}
}

func (s *CycleController) _suspendConveyorAndWait(wg *sync.WaitGroup) {
	s.debugLock.Lock()
	defer s.debugLock.Unlock()

	if s.cycleFn != nil {
		panic(throw.IllegalState())
	}

	if s.suspend == nil {
		s.suspend = make(synckit.ClosableSignalChannel)
	}

	flags := s.debugFlags.Load()
	if flags&isNotScanning != 0 {
		wg.Done()
		return
	}

	s.debugFlags.SetBits(callOnce)
	s.cycleFn = func(bool, bool) {
		wg.Done()
	}
}

func (s *CycleController) SuspendConveyorAndWait() {
	wg := sync.WaitGroup{}
	wg.Add(1)

	s._suspendConveyorAndWait(&wg)

	wg.Wait()
}

func (s *CycleController) SuspendConveyorAndWaitThenResetActive() {
	s.SuspendConveyorAndWait()
	s.ResetActiveConveyorFlag()
}

func (s *CycleController) ResumeConveyor() {
	s.debugLock.Lock()
	defer s.debugLock.Unlock()
	s._resumeConveyor()
}

func (s *CycleController) _resumeConveyor() {
	if ch := s.suspend; ch != nil {
		s.suspend = nil
		close(ch)
	}
}

func (s *CycleController) ResetActiveConveyorFlag() {
	s.debugLock.Lock()
	defer s.debugLock.Unlock()
	s.debugFlags.UnsetBits(hasActive)
}

const (
	hasActive     = 1
	isIdle        = 2
	isNotScanning = 4
	callOnce      = 8
)

func (s *CycleController) onCycleUpdate(fn func() uint32) synckit.SignalChannel {
	updateFn, ch := s._onCycleUpdate(fn)
	if updateFn != nil {
		updateFn()
	}
	return ch
}

func (s *CycleController) _onCycleUpdate(fn func() uint32) (func(), synckit.SignalChannel) {
	s.debugLock.Lock()
	defer s.debugLock.Unlock()

	// makes sure that calling Wait in a parallel thread will not lock up caller of Suspend
	if cs := s.debugFlags.Load(); cs&callOnce != 0 {
		s.debugFlags.UnsetBits(callOnce)
		if cycleFn := s.cycleFn; cycleFn != nil {
			s.cycleFn = nil
			go cycleFn(cs&hasActive != 0, cs&isIdle != 0)
		}
	}

	cs := fn()
	if cycleFn := s.cycleFn; cycleFn != nil {
		return func() {
			cycleFn(cs&hasActive != 0, cs&isIdle != 0)
		}, s.suspend
	}

	return nil, s.suspend
}

func (s *CycleController) onConveyorCycle(state conveyor.CycleState) {
	ch := s.onCycleUpdate(func() uint32 {
		switch state {
		case conveyor.ScanActive:
			return s.debugFlags.SetBits(hasActive | isNotScanning)
		case conveyor.ScanIdle:
			return s.debugFlags.SetBits(isIdle | isNotScanning)
		case conveyor.Scanning:
			return s.debugFlags.UnsetBits(isIdle | isNotScanning)
		default:
			panic(throw.Impossible())
		}
	})
	if ch != nil && state == conveyor.Scanning {
		<-ch
	}
}

func (s *CycleController) setWaitCallback(cycleFn cycleFunc) {
	s.onCycleUpdate(func() uint32 {
		if cycleFn != nil {
			s._resumeConveyor()
		} else {
			s.debugFlags.UnsetBits(hasActive)
		}
		s.cycleFn = cycleFn
		return s.debugFlags.Load()
	})
}

