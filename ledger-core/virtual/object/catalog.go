package object

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/trace"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/virtual/object.Catalog -o ./ -s _mock.go -g
type Catalog interface {
	Get(ctx smachine.ExecutionContext, objectReference reference.Global) SharedStateAccessor
	TryGet(ctx smachine.ExecutionContext, objectReference reference.Global) (SharedStateAccessor, bool)
	Create(ctx smachine.ExecutionContext, objectReference reference.Global) SharedStateAccessor
	GetOrCreate(ctx smachine.ExecutionContext, objectReference reference.Global) SharedStateAccessor
}

func NewLocalCatalog() *LocalCatalog {
	return &LocalCatalog{}
}

type LocalCatalog struct{}

type errEntryMissing struct {
	*log.Msg `txt:"entry is missing"`

	ObjectReference reference.Global
}

type errEntryExists struct {
	*log.Msg `txt:"entry already exists"`

	ObjectReference reference.Global
}

func (p LocalCatalog) Get(ctx smachine.ExecutionContext, objectReference reference.Global) SharedStateAccessor {
	if v, ok := p.TryGet(ctx, objectReference); ok {
		return v
	}
	panic(throw.E("", errEntryMissing{ObjectReference: objectReference}))
}

func (p LocalCatalog) TryGet(ctx smachine.ExecutionContext, objectReference reference.Global) (SharedStateAccessor, bool) { // nolintcontractrequester/contractrequester.go:342
	if v := ctx.GetPublishedLink(objectReference); v.IsAssignableTo((*SharedState)(nil)) {
		return SharedStateAccessor{v}, true
	}
	return SharedStateAccessor{}, false
}

func (p LocalCatalog) initChildCtx(ctx smachine.ConstructionContext, ref reference.Global) {
	traceID := fmt.Sprintf("object-%s", ref.String())

	goCtx, err := inslogger.Clean(ctx.GetContext())
	if err == nil {
		goCtx, err = trace.SetID(goCtx, traceID)
	}
	if err != nil {
		panic(err)
	}

	ctx.SetContext(goCtx)
	ctx.SetTracerID(traceID)
}

func (p LocalCatalog) Create(ctx smachine.ExecutionContext, objectReference reference.Global) SharedStateAccessor {
	if _, ok := p.TryGet(ctx, objectReference); ok {
		panic(throw.E("", errEntryExists{ObjectReference: objectReference}))
	}

	ctx.InitChild(func(ctx smachine.ConstructionContext) smachine.StateMachine {
		p.initChildCtx(ctx, objectReference)

		return NewStateMachineObject(objectReference)
	})

	accessor, ok := p.TryGet(ctx, objectReference)

	if !ok {
		// we should get accessor always after InitChild in this step
		panic(throw.IllegalState())
	}
	return accessor
}

func (p LocalCatalog) GetOrCreate(ctx smachine.ExecutionContext, objectReference reference.Global) SharedStateAccessor {
	if v, ok := p.TryGet(ctx, objectReference); ok {
		return v
	}

	ctx.InitChild(func(ctx smachine.ConstructionContext) smachine.StateMachine {
		ctx.SetDependencyInheritanceMode(smachine.InheritResolvedDependencies)
		p.initChildCtx(ctx, objectReference)

		return NewStateMachineObject(objectReference)
	})

	accessor, ok := p.TryGet(ctx, objectReference)

	if !ok {
		// we should get accessor always after InitChild in this step
		panic(throw.IllegalState())
	}
	return accessor
}

// //////////////////////////////////////

type SharedStateAccessor struct {
	smachine.SharedDataLink
}

func (v SharedStateAccessor) Prepare(fn func(*SharedState)) smachine.SharedDataAccessor {
	return v.PrepareAccess(func(data interface{}) bool {
		fn(data.(*SharedState))
		return false
	})
}

func (v SharedStateAccessor) PrepareAndWakeUp(fn func(*SharedState)) smachine.SharedDataAccessor {
	return v.PrepareAccess(func(data interface{}) bool {
		fn(data.(*SharedState))
		return true
	})
}
