// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package recordchecker

import (
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type SerializableBasicRecord interface {
	rmsreg.GoGoSerializable
	rms.BasicRecord
}

func recordToAnyRecordLazy(rec SerializableBasicRecord) rms.AnyRecordLazy {
	if rec == nil {
		panic(throw.IllegalValue())
	}
	rv := rms.AnyRecordLazy{}
	if err := rv.SetAsLazy(rec); err != nil {
		panic(err)
	}
	return rv
}

func TestChecker_HappyPath(t *testing.T) {
	var (
		mc                = minimock.NewController(t)
		objARef           = rms.NewReference(gen.UniqueGlobalRef())
		constructorA      = rms.NewReference(gen.UniqueGlobalRef())
		outgoingA         = rms.NewReference(gen.UniqueGlobalRef())
		outgoingResponseA = rms.NewReference(gen.UniqueGlobalRef())
		methodA           = rms.NewReference(gen.UniqueGlobalRef())
		lineMemARef       = rms.NewReference(gen.UniqueGlobalRef())
		lineActivateARef  = rms.NewReference(gen.UniqueGlobalRef())
		objBRef           = rms.NewReference(gen.UniqueGlobalRef())
		constructorB      = rms.NewReference(gen.UniqueGlobalRef())
		outgoingB         = rms.NewReference(gen.UniqueGlobalRef())
		outgoingResponseB = rms.NewReference(gen.UniqueGlobalRef())
	)

	checker := NewLMNMessageChecker(mc)
	{ // object A
		chain := checker.NewChain(objARef)
		lineInboundConstructor := chain.AddRootMessage(
			&rms.RLifelineStart{},
			nil,
		).AddChild(
			&rms.RLineInboundRequest{},
			nil,
		)
		lineInboundConstructor.AddChild(
			&rms.ROutboundRequest{},
			nil,
		).AddChild(
			&rms.ROutboundResponse{},
			nil,
		).AddChild(
			&rms.RInboundResponse{},
			nil,
		)
		lineInbound := lineInboundConstructor.AddChild(
			&rms.RLineMemory{},
			nil,
		).AddChild(
			&rms.RLineActivate{},
			nil,
		).AddChild(
			&rms.RLineInboundRequest{},
			nil,
		)
		lineInbound.AddChild(
			&rms.RInboundResponse{},
			nil,
		)
		lineInbound.AddChild(
			&rms.RLineMemory{},
			nil,
		)
	}
	{ // object B
		chain := checker.NewChain(objBRef)
		lineInboundConstructor := chain.AddRootMessage(
			&rms.RLifelineStart{},
			nil,
		).AddChild(
			&rms.RLineInboundRequest{},
			nil,
		)
		lineInboundConstructor.AddChild(
			&rms.ROutboundRequest{},
			nil,
		).AddChild(
			&rms.ROutboundResponse{},
			nil,
		).AddChild(
			&rms.RInboundResponse{},
			nil,
		)
		lineInboundConstructor.AddChild(
			&rms.RLineMemory{},
			nil,
		)
	}

	{ // constructor A with outgoing
		require.NoError(t, checker.ProcessMessage(rms.LRegisterRequest{
			AnyRecordLazy:  recordToAnyRecordLazy(&rms.RLifelineStart{}),
			AnticipatedRef: objARef,
		}))
		require.NoError(t, checker.ProcessMessage(rms.LRegisterRequest{
			AnyRecordLazy: recordToAnyRecordLazy(
				&rms.RLineInboundRequest{
					RootRef: objARef,
					PrevRef: objARef,
				}),
			AnticipatedRef: constructorA,
		}))
		require.NoError(t, checker.ProcessMessage(rms.LRegisterRequest{
			AnyRecordLazy: recordToAnyRecordLazy(
				&rms.ROutboundRequest{
					RootRef: objARef,
					PrevRef: constructorA,
				}),
			AnticipatedRef: outgoingA,
		}))
		require.NoError(t, checker.ProcessMessage(rms.LRegisterRequest{
			AnyRecordLazy: recordToAnyRecordLazy(
				&rms.ROutboundResponse{
					RootRef: objARef,
					PrevRef: outgoingA,
				}),
			AnticipatedRef: outgoingResponseA,
		}))
		require.NoError(t, checker.ProcessMessage(rms.LRegisterRequest{
			AnyRecordLazy: recordToAnyRecordLazy(
				&rms.RInboundResponse{
					RootRef: objARef,
					PrevRef: outgoingResponseA,
				}),
			AnticipatedRef: rms.NewReference(gen.UniqueGlobalRef()),
		}))
		require.NoError(t, checker.ProcessMessage(rms.LRegisterRequest{
			AnyRecordLazy: recordToAnyRecordLazy(
				&rms.RLineMemory{
					RootRef: objARef,
					PrevRef: constructorA,
				}),
			AnticipatedRef: lineMemARef,
		}))
		require.NoError(t, checker.ProcessMessage(rms.LRegisterRequest{
			AnyRecordLazy: recordToAnyRecordLazy(
				&rms.RLineActivate{
					RootRef: objARef,
					PrevRef: lineMemARef,
				}),
			AnticipatedRef: lineActivateARef,
		}))
	}
	{ // constructor B with call A method
		require.NoError(t, checker.ProcessMessage(rms.LRegisterRequest{
			AnyRecordLazy:  recordToAnyRecordLazy(&rms.RLifelineStart{}),
			AnticipatedRef: objBRef,
		}))
		require.NoError(t, checker.ProcessMessage(rms.LRegisterRequest{
			AnyRecordLazy: recordToAnyRecordLazy(
				&rms.RLineInboundRequest{
					RootRef: objBRef,
					PrevRef: objBRef,
				}),
			AnticipatedRef: constructorB,
		}))
		require.NoError(t, checker.ProcessMessage(rms.LRegisterRequest{
			AnyRecordLazy: recordToAnyRecordLazy(
				&rms.ROutboundRequest{
					RootRef: objBRef,
					PrevRef: constructorB,
				}),
			AnticipatedRef: outgoingB,
		}))
	}
	{ // call A method
		require.NoError(t, checker.ProcessMessage(rms.LRegisterRequest{
			AnyRecordLazy: recordToAnyRecordLazy(
				&rms.RLineInboundRequest{
					RootRef: objARef,
					PrevRef: lineActivateARef,
				}),
			AnticipatedRef: methodA,
		}))
		require.NoError(t, checker.ProcessMessage(rms.LRegisterRequest{
			AnyRecordLazy: recordToAnyRecordLazy(
				&rms.RInboundResponse{
					RootRef: objARef,
					PrevRef: methodA,
				}),
			AnticipatedRef: rms.NewReference(gen.UniqueGlobalRef()),
		}))
		require.NoError(t, checker.ProcessMessage(rms.LRegisterRequest{
			AnyRecordLazy: recordToAnyRecordLazy(
				&rms.RLineMemory{
					RootRef: objARef,
					PrevRef: methodA,
				}),
			AnticipatedRef: rms.NewReference(gen.UniqueGlobalRef()),
		}))
	}
	{ // done constructor B
		require.NoError(t, checker.ProcessMessage(rms.LRegisterRequest{
			AnyRecordLazy: recordToAnyRecordLazy(
				&rms.ROutboundResponse{
					RootRef: objBRef,
					PrevRef: outgoingB,
				}),
			AnticipatedRef: outgoingResponseB,
		}))
		require.NoError(t, checker.ProcessMessage(rms.LRegisterRequest{
			AnyRecordLazy: recordToAnyRecordLazy(
				&rms.RInboundResponse{
					RootRef: objBRef,
					PrevRef: outgoingResponseB,
				}),
			AnticipatedRef: rms.NewReference(gen.UniqueGlobalRef()),
		}))
		require.NoError(t, checker.ProcessMessage(rms.LRegisterRequest{
			AnyRecordLazy: recordToAnyRecordLazy(
				&rms.RLineMemory{
					RootRef: objBRef,
					PrevRef: constructorB,
				}),
			AnticipatedRef: rms.NewReference(gen.UniqueGlobalRef()),
		}))
	}

	mc.Finish()
}

func TestChecker_UnregisteredMessage(t *testing.T) {
	var (
		mc     = minimock.NewController(t)
		objRef = rms.NewReference(gen.UniqueGlobalRef())
	)

	checker := NewLMNMessageChecker(mc)
	chain := checker.NewChain(objRef)
	chain.AddRootMessage(
		&rms.RLifelineStart{},
		nil,
	)
	require.NoError(t, checker.ProcessMessage(rms.LRegisterRequest{
		AnyRecordLazy:  recordToAnyRecordLazy(&rms.RLifelineStart{}),
		AnticipatedRef: objRef,
	}))
	require.Error(t, checker.ProcessMessage(rms.LRegisterRequest{
		AnyRecordLazy: recordToAnyRecordLazy(
			&rms.RInboundRequest{
				RootRef: objRef,
				PrevRef: objRef,
			}),
		AnticipatedRef: rms.NewReference(gen.UniqueGlobalRef()),
	}))
	mc.Finish()
}
