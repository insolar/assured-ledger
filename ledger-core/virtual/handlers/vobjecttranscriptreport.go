// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate sm-uml-gen -f $GOFILE

package handlers

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/memorycache"
	memoryCacheAdapter "github.com/insolar/assured-ledger/ledger-core/virtual/memorycache/adapter"
)

type SMVObjectTranscriptReport struct {
	// input arguments
	Meta    *payload.Meta
	Payload *rms.VObjectTranscriptReport

	memoryCache   memoryCacheAdapter.MemoryCache

	entryIndex int

	objDesc descriptor.Object
	incomingRequest *rms.VObjectTranscriptReport_TranscriptEntryIncomingRequest
}

/* -------- Declaration ------------- */

var dSMVObjectTranscriptReportInstance smachine.StateMachineDeclaration = &dSMVObjectTranscriptReport{}

type dSMVObjectTranscriptReport struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMVObjectTranscriptReport) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	s := sm.(*SMVObjectTranscriptReport)

	injector.MustInject(&s.memoryCache)
}

func (*dSMVObjectTranscriptReport) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMVObjectTranscriptReport)
	return s.Init
}

/* -------- Instance ------------- */

func (s *SMVObjectTranscriptReport) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMVObjectTranscriptReportInstance
}

func (s *SMVObjectTranscriptReport) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepProcess)
}

func (s *SMVObjectTranscriptReport) stepProcess(ctx smachine.ExecutionContext) smachine.StateUpdate {

	entries := s.Payload.ObjectTranscript.GetEntries()
	entry := entries[s.entryIndex].Get()

	switch tEntry := entry.(type) {
	case *rms.VObjectTranscriptReport_TranscriptEntryIncomingRequest:
		s.incomingRequest = tEntry
		return ctx.Jump(s.stepIncomingRequest)

	}

	return ctx.Stop()
}

func (s *SMVObjectTranscriptReport) stepIncomingRequest(ctx smachine.ExecutionContext) smachine.StateUpdate {
	ref := s.incomingRequest.ObjectMemory.GetGlobal()

	done := false
	s.memoryCache.PrepareAsync(ctx, func(ctx context.Context, svc memorycache.Service) smachine.AsyncResultFunc {
		obj, err := svc.Get(ctx, ref)
		return func(ctx smachine.AsyncResultContext) {
			defer func() { done = true }()
			s.objDesc = obj
			if err != nil {
				ctx.Log().Error("failed to get memory", err)
			}
		}
	}).Start()

	return ctx.Sleep().ThenJump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
		if !done {
			return ctx.Sleep().ThenRepeat()
		}
		if s.objDesc == nil {
			return ctx.Jump(s.stepRequestMemory)
		}
		return ctx.Jump(s.stepRequestMemory)

	})
}

func (s *SMVObjectTranscriptReport) stepRequestMemory(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Stop()
}
