package smachines

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/store"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
)

const RecordBatchThreshold = 1000

type sharedDropBatch struct {
	records      []*store.Record
	recordNumber int
	syncFinished smachine.SyncLink
}

func (b *sharedDropBatch) appendRecords(recs []*store.Record) {
	b.records = append(b.records, recs...)
	b.recordNumber += len(recs)
}

/* -------- Declaration ------------- */

var declDropBatch smachine.StateMachineDeclaration = &declarationDropBatch{}

type declarationDropBatch struct {
	smachine.StateMachineDeclTemplate
}

func (*declarationDropBatch) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	m := sm.(*DropBatch)
	injector.MustInject(&m.syncAdapter)
	injector.MustInject(&m.hashingAdapter)
	injector.MustInject(&m.pulseSlot)
	injector.MustInject(&m.pcs)
	injector.MustInject(&m.records)
}

func (*declarationDropBatch) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	return sm.(*DropBatch).Init
}

/* -------- Instance ------------- */

type DropBatch struct {
	jetID insolar.JetID

	hashingAdapter *HashingAdapter
	syncAdapter    *SyncAdapter
	pulseSlot      *conveyor.PulseSlot
	pcs            insolar.PlatformCryptographyScheme
	records        *store.RecordStore

	ownedDropBatch sharedDropBatch
	syncFinished   smachine.BoolConditionalLink
}

func NewDropBatch(jetID insolar.JetID) *DropBatch {
	return &DropBatch{jetID: jetID}
}

func (s *DropBatch) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return declDropBatch
}

func (s *DropBatch) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	s.syncFinished = smachine.NewConditionalBool(false, "syncFinished")
	s.ownedDropBatch.syncFinished = s.syncFinished.SyncLink()

	link := ctx.Share(&s.ownedDropBatch, smachine.ShareDataWakesUpAfterUse)
	if !ctx.Publish(s.jetID, link) {
		return ctx.Stop()
	}

	return ctx.Jump(s.waitForBatch)
}

func (s *DropBatch) waitForBatch(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if len(s.ownedDropBatch.records) < RecordBatchThreshold {
		return ctx.Sleep().ThenRepeat()
	}
	return ctx.Jump(s.calculateHashes)
}

func (s *DropBatch) calculateHashes(ctx smachine.ExecutionContext) smachine.StateUpdate {
	ctx.UnpublishAll()

	return s.hashingAdapter.PrepareAsync(ctx, func() smachine.AsyncResultFunc {
		pulseNumber := s.pulseSlot.PulseData().GetPulseNumber()
		calculatedIDs := make([]insolar.ID, len(s.ownedDropBatch.records))

		for i, r := range s.ownedDropBatch.records {
			buf, err := r.Record.Marshal()
			if err != nil {
				panic(err)
			}
			h := s.pcs.ReferenceHasher()
			h.Hash(buf)
			calculatedIDs[i] = *insolar.NewID(pulseNumber, h.Sum(nil))
		}
		return func(ctx smachine.AsyncResultContext) {
			for i, r := range s.ownedDropBatch.records {
				r.ID = calculatedIDs[i]
			}
		}
	}).DelayedStart().ThenJump(s.sync)
}

func (s *DropBatch) sync(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return s.syncAdapter.PrepareAsync(ctx, func() smachine.AsyncResultFunc {
		s.records.Store(s.ownedDropBatch.records)
		// TODO: network sync.
		return nil
	}).DelayedStart().ThenJump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
		ctx.ApplyAdjustment(s.syncFinished.NewValue(true))
		return ctx.Stop()
	})
}

func sharedDropBatchLink(ctx smachine.ExecutionContext, jetID insolar.JetID) smachine.SharedDataLink {
	link := ctx.GetPublishedLink(jetID)
	if link.IsZero() {
		ctx.InitChild(func(ctx smachine.ConstructionContext) smachine.StateMachine {
			return NewDropBatch(jetID)
		})
		link = ctx.GetPublishedLink(jetID)
		if link.IsZero() {
			panic("failed to acquire shared drop")
		}
	}
	return link
}
