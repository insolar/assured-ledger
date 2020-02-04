package smachines

import (
	"context"
	"fmt"
	"sync"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/bus/meta"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/adapters"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
)

type Worker struct {
	conveyor *conveyor.PulseConveyor
	stopped  sync.WaitGroup
}

func NewWorker() Worker {
	return Worker{}
}

func (w *Worker) Stop() {
	w.conveyor.StopNoWait()
	w.stopped.Wait()
}

func (w *Worker) AttachTo(conveyor *conveyor.PulseConveyor) {
	if conveyor == nil {
		panic("illegal value")
	}
	if w.conveyor != nil {
		panic("illegal state")
	}
	w.conveyor = conveyor
	w.stopped.Add(1)
	conveyor.StartWorker(nil, func() {
		w.stopped.Done()
	})
}

type Dispatcher struct {
	conveyor      *conveyor.PulseConveyor
	previousPulse insolar.PulseNumber
}

func NewDispatcher(conveyor *conveyor.PulseConveyor) *Dispatcher {
	return &Dispatcher{conveyor: conveyor}
}

func (d *Dispatcher) BeginPulse(_ context.Context, pulseObject insolar.Pulse) {
	if err := d.conveyor.CommitPulseChange(adapters.NewPulseData(pulseObject).AsRange()); err != nil {
		panic(err)
	}
}

func (d *Dispatcher) ClosePulse(_ context.Context, pulseObject insolar.Pulse) {
	d.previousPulse = pulseObject.PulseNumber
}

func (d *Dispatcher) Process(msg *message.Message) error {
	plMeta := payload.Meta{}
	err := plMeta.Unmarshal(msg.Payload)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal payload.Meta")
	}
	ctx, _ := inslogger.WithTraceField(context.Background(), msg.Metadata.Get(meta.TraceID))
	return d.conveyor.AddInput(ctx, plMeta.Pulse, msg)
}

func CommonFactory(_ pulse.Number, input conveyor.InputEvent) smachine.CreateFunc {
	switch i := input.(type) {
	case *message.Message:
		return MessageFactory(i)
	default:
		panic(fmt.Sprintf("unknown event type, got %T", input))
	}
}

func MessageFactory(message *message.Message) smachine.CreateFunc {
	messageMeta := payload.Meta{}
	err := messageMeta.Unmarshal(message.Payload)
	if err != nil {
		panic(errors.Wrap(err, "failed to unmarshal meta"))
	}

	payloadType, err := payload.UnmarshalType(messageMeta.Payload)
	if err != nil {
		panic(fmt.Sprintf("failed to unmarshal payload type: %s", err.Error()))
	}

	switch payloadType {
	case payload.TypeV2SetRequestResult:
		return func(ctx smachine.ConstructionContext) smachine.StateMachine {
			return NewSetResult(messageMeta)
		}
	default:
		panic("unsupported payload type")
	}
}
