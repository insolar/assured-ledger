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

	checker := NewChecker(mc)
	{ // object A
		lineInboundConstructor := checker.CreateChainFromRLifeline(
			rms.LRegisterRequest{
				AnyRecordLazy:  recordToAnyRecordLazy(&rms.RLifelineStart{}),
				AnticipatedRef: objARef,
			},
			nil,
		).AddMessage(
			rms.LRegisterRequest{
				AnyRecordLazy: recordToAnyRecordLazy(&rms.RLineInboundRequest{}),
			},
			nil,
		)
		lineInboundConstructor.AddMessage(
			rms.LRegisterRequest{
				AnyRecordLazy: recordToAnyRecordLazy(&rms.ROutboundRequest{}),
			},
			nil,
		).AddMessage(
			rms.LRegisterRequest{
				AnyRecordLazy: recordToAnyRecordLazy(&rms.ROutboundResponse{}),
			},
			nil,
		).AddMessage(
			rms.LRegisterRequest{
				AnyRecordLazy: recordToAnyRecordLazy(&rms.RInboundResponse{}),
			},
			nil,
		)
		lineInbound := lineInboundConstructor.AddMessage(
			rms.LRegisterRequest{
				AnyRecordLazy: recordToAnyRecordLazy(&rms.RLineMemory{}),
			},
			nil,
		).AddMessage(
			rms.LRegisterRequest{
				AnyRecordLazy: recordToAnyRecordLazy(&rms.RLineActivate{}),
			},
			nil,
		).AddMessage(
			rms.LRegisterRequest{
				AnyRecordLazy: recordToAnyRecordLazy(&rms.RLineInboundRequest{}),
			},
			nil,
		)
		lineInbound.AddMessage(
			rms.LRegisterRequest{
				AnyRecordLazy: recordToAnyRecordLazy(&rms.RInboundResponse{}),
			},
			nil,
		)
		lineInbound.AddMessage(
			rms.LRegisterRequest{
				AnyRecordLazy: recordToAnyRecordLazy(&rms.RLineMemory{}),
			},
			nil,
		)
	}
	{ // object B
		lineInboundConstructor := checker.CreateChainFromRLifeline(
			rms.LRegisterRequest{
				AnyRecordLazy:  recordToAnyRecordLazy(&rms.RLifelineStart{}),
				AnticipatedRef: objBRef,
			},
			nil,
		).AddMessage(
			rms.LRegisterRequest{
				AnyRecordLazy: recordToAnyRecordLazy(&rms.RLineInboundRequest{}),
			},
			nil,
		)
		lineInboundConstructor.AddMessage(
			rms.LRegisterRequest{
				AnyRecordLazy: recordToAnyRecordLazy(&rms.ROutboundRequest{}),
			},
			nil,
		).AddMessage(
			rms.LRegisterRequest{
				AnyRecordLazy: recordToAnyRecordLazy(&rms.ROutboundResponse{}),
			},
			nil,
		).AddMessage(
			rms.LRegisterRequest{
				AnyRecordLazy: recordToAnyRecordLazy(&rms.RInboundResponse{}),
			},
			nil,
		)
		lineInboundConstructor.AddMessage(
			rms.LRegisterRequest{
				AnyRecordLazy: recordToAnyRecordLazy(&rms.RLineMemory{}),
			},
			nil,
		)
	}

	{ // constructor A with outgoing
		var (
			chainAChecker = checker.GetChainValidatorList().GetChainValidatorByReference(objARef.GetValue())
			err           error
			messages      = []rms.LRegisterRequest{
				{
					AnyRecordLazy:  recordToAnyRecordLazy(&rms.RLifelineStart{}),
					AnticipatedRef: objARef,
				},
				{
					AnyRecordLazy: recordToAnyRecordLazy(
						&rms.RLineInboundRequest{
							RootRef: objARef,
							PrevRef: objARef,
						}),
					AnticipatedRef: constructorA,
				},
				{
					AnyRecordLazy: recordToAnyRecordLazy(
						&rms.ROutboundRequest{
							RootRef: objARef,
							PrevRef: constructorA,
						}),
					AnticipatedRef: outgoingA,
				},
				{
					AnyRecordLazy: recordToAnyRecordLazy(
						&rms.ROutboundResponse{
							RootRef: objARef,
							PrevRef: outgoingA,
						}),
					AnticipatedRef: outgoingResponseA,
				},
				{
					AnyRecordLazy: recordToAnyRecordLazy(
						&rms.RInboundResponse{
							RootRef: objARef,
							PrevRef: outgoingResponseA,
						}),
					AnticipatedRef: rms.NewReference(gen.UniqueGlobalRef()),
				},
				{
					AnyRecordLazy: recordToAnyRecordLazy(
						&rms.RLineMemory{
							RootRef: objARef,
							PrevRef: constructorA,
						}),
					AnticipatedRef: lineMemARef,
				},
				{
					AnyRecordLazy: recordToAnyRecordLazy(
						&rms.RLineActivate{
							RootRef: objARef,
							PrevRef: lineMemARef,
						}),
					AnticipatedRef: lineActivateARef,
				},
			}
		)

		for _, msg := range messages {
			_, err = chainAChecker.Feed(msg)
			require.NoError(t, err)
		}
	}
	{ // constructor B with call A method
		var (
			chainBChecker = checker.GetChainValidatorList().GetChainValidatorByReference(objBRef.GetValue())
			err           error
			messages      = []rms.LRegisterRequest{
				{
					AnyRecordLazy:  recordToAnyRecordLazy(&rms.RLifelineStart{}),
					AnticipatedRef: objBRef,
				},
				{
					AnyRecordLazy: recordToAnyRecordLazy(
						&rms.RLineInboundRequest{
							RootRef: objBRef,
							PrevRef: objBRef,
						}),
					AnticipatedRef: constructorB,
				},
				{
					AnyRecordLazy: recordToAnyRecordLazy(
						&rms.ROutboundRequest{
							RootRef: objBRef,
							PrevRef: constructorB,
						}),
					AnticipatedRef: outgoingB,
				},
			}
		)
		for _, msg := range messages {
			chainBChecker, err = chainBChecker.Feed(msg)
			require.NoError(t, err)
		}
	}
	{ // call A method
		var (
			chainAChecker = checker.GetChainValidatorList().GetChainValidatorByReference(objARef.GetValue())
			err           error
			messages      = []rms.LRegisterRequest{
				{
					AnyRecordLazy: recordToAnyRecordLazy(
						&rms.RLineInboundRequest{
							RootRef: objARef,
							PrevRef: lineActivateARef,
						}),
					AnticipatedRef: methodA,
				},
				{
					AnyRecordLazy: recordToAnyRecordLazy(
						&rms.RInboundResponse{
							RootRef: objARef,
							PrevRef: methodA,
						}),
					AnticipatedRef: rms.NewReference(gen.UniqueGlobalRef()),
				},
				{
					AnyRecordLazy: recordToAnyRecordLazy(
						&rms.RLineMemory{
							RootRef: objARef,
							PrevRef: methodA,
						}),
					AnticipatedRef: rms.NewReference(gen.UniqueGlobalRef()),
				},
			}
		)
		for _, msg := range messages {
			_, err = chainAChecker.Feed(msg)
			require.NoError(t, err)
		}
	}
	{ // done constructor B
		var (
			chainBChecker = checker.GetChainValidatorList().GetChainValidatorByReference(objBRef.GetValue())
			err           error
			messages      = []rms.LRegisterRequest{
				{
					AnyRecordLazy: recordToAnyRecordLazy(
						&rms.ROutboundResponse{
							RootRef: objBRef,
							PrevRef: outgoingB,
						}),
					AnticipatedRef: outgoingResponseB,
				},
				{
					AnyRecordLazy: recordToAnyRecordLazy(
						&rms.RInboundResponse{
							RootRef: objBRef,
							PrevRef: outgoingResponseB,
						}),
					AnticipatedRef: rms.NewReference(gen.UniqueGlobalRef()),
				},
				{
					AnyRecordLazy: recordToAnyRecordLazy(
						&rms.RLineMemory{
							RootRef: objBRef,
							PrevRef: constructorB,
						}),
					AnticipatedRef: rms.NewReference(gen.UniqueGlobalRef()),
				},
			}
		)
		for _, msg := range messages {
			_, err = chainBChecker.Feed(msg)
			require.NoError(t, err)
		}
	}

	mc.Finish()
}

func TestChecker_UnregisteredMessage(t *testing.T) {
	var (
		mc           = minimock.NewController(t)
		objRef       = rms.NewReference(gen.UniqueGlobalRef())
		chainChecker ChainValidator
		err          error
	)

	checker := NewChecker(mc)
	chain := checker.CreateChainFromReference(objRef)
	chain.AddMessage(
		rms.LRegisterRequest{
			AnyRecordLazy: recordToAnyRecordLazy(&rms.RLifelineStart{}),
		},
		nil,
	)
	chainChecker = checker.GetChainValidatorList().GetChainValidatorByReference(objRef.GetValue())
	chainChecker, err = chainChecker.Feed(rms.LRegisterRequest{
		AnyRecordLazy:  recordToAnyRecordLazy(&rms.RLifelineStart{}),
		AnticipatedRef: objRef,
	})
	require.NoError(t, err)
	chainChecker, err = chainChecker.Feed(rms.LRegisterRequest{
		AnyRecordLazy: recordToAnyRecordLazy(
			&rms.RInboundRequest{
				RootRef: objRef,
				PrevRef: objRef,
			}),
		AnticipatedRef: rms.NewReference(gen.UniqueGlobalRef()),
	})
	require.Error(t, err)
	mc.Finish()
}

func TestChecker_UnsentMessage(t *testing.T) {
	var (
		objRef       = rms.NewReference(gen.UniqueGlobalRef())
		chainChecker ChainValidator
		err          error
	)

	checker := NewChecker(nil)
	chain := checker.CreateChainFromReference(objRef)
	chain.AddMessage(
		rms.LRegisterRequest{
			AnyRecordLazy: recordToAnyRecordLazy(&rms.RLifelineStart{}),
		},
		nil,
	).AddMessage(
		rms.LRegisterRequest{
			AnyRecordLazy: recordToAnyRecordLazy(&rms.RInboundRequest{}),
		},
		nil,
	)
	chainChecker = checker.GetChainValidatorList().GetChainValidatorByReference(objRef.GetValue())
	chainChecker, err = chainChecker.Feed(rms.LRegisterRequest{
		AnyRecordLazy:  recordToAnyRecordLazy(&rms.RLifelineStart{}),
		AnticipatedRef: objRef,
	})
	require.NoError(t, err)

	require.False(t, checker.GetChainValidatorList().IsFinished())
}
