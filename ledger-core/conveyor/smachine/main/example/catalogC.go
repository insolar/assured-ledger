//go:generate sm-uml-gen -f $GOFILE

package example

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

func CreateCatalogC() CatalogC { // nolint:golint
	return &catalogC{}
}

type CatalogC = *catalogC

type CustomSharedState struct {
	key     longbits.ByteString
	Mutex   smachine.SyncLink
	Text    string
	Counter int
}

func (p *CustomSharedState) GetKey() longbits.ByteString {
	return p.key
}

type CustomSharedStateAccessor struct {
	link smachine.SharedDataLink
}

func (v CustomSharedStateAccessor) Prepare(fn func(*CustomSharedState)) smachine.SharedDataAccessor {
	return v.link.PrepareAccess(func(data interface{}) bool {
		fn(data.(*CustomSharedState))
		return false
	})
}

type catalogC struct {
}

func (p *catalogC) Get(ctx smachine.ExecutionContext, key longbits.ByteString) CustomSharedStateAccessor {
	if v, ok := p.TryGet(ctx, key); ok {
		return v
	}
	panic(fmt.Sprintf("missing entry: %s", key))
}

func (p *catalogC) TryGet(ctx smachine.ExecutionContext, key longbits.ByteString) (CustomSharedStateAccessor, bool) {

	if v := ctx.GetPublishedLink(key); v.IsAssignableTo((*CustomSharedState)(nil)) {
		return CustomSharedStateAccessor{v}, true
	}
	return CustomSharedStateAccessor{}, false
}

func (p *catalogC) GetOrCreate(ctx smachine.ExecutionContext, key longbits.ByteString) CustomSharedStateAccessor {
	if v, ok := p.TryGet(ctx, key); ok {
		return v
	}

	ctx.InitChild(func(ctx smachine.ConstructionContext) smachine.StateMachine {
		return &catalogEntryCSM{sharedState: CustomSharedState{
			key: key,
			//Mutex: smsync.NewExclusiveWithFlags("", 0), //smachine.QueueAllowsPriority),
			Mutex: smsync.NewSemaphoreWithFlags(3, "", smsync.QueueAllowsPriority).SyncLink(),
		}}
	})

	return p.Get(ctx, key)
}

type catalogEntryCSM struct {
	smachine.StateMachineDeclTemplate
	sharedState CustomSharedState
}

func (sm *catalogEntryCSM) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return sm.Init
}

func (sm *catalogEntryCSM) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return sm
}

func (sm *catalogEntryCSM) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	sdl := ctx.Share(&sm.sharedState, 0)
	if !ctx.Publish(sm.sharedState.key, sdl) {
		return ctx.Stop()
	}
	return ctx.JumpExt(smachine.SlotStep{Transition: sm.State1, Flags: smachine.StepWeak})
}

func (sm *catalogEntryCSM) State1(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Sleep().ThenRepeat()
}
