package v2

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/record"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/smachines"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/store"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/statemachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
)

type mockSender struct {
	wg sync.WaitGroup
}

func newMockSender() *mockSender {
	return &mockSender{}
}

func (m *mockSender) SendRole(
	ctx context.Context, msg *message.Message, role insolar.DynamicRole, object insolar.Reference,
) (<-chan *message.Message, func()) {
	return nil, func() {}
}

func (m *mockSender) SendTarget(ctx context.Context, msg *message.Message, target insolar.Reference) (<-chan *message.Message, func()) {
	return nil, func() {}
}

// State machine will call this as a response. When all machines reply, we can assume that all the work is done.
func (m *mockSender) Reply(ctx context.Context, origin payload.Meta, reply *message.Message) {
	m.wg.Done()
}

func (m *mockSender) init(replies int) {
	m.wg.Add(replies)
}

func (m *mockSender) wait() {
	m.wg.Wait()
}

type components struct {
	dispatcher *smachines.Dispatcher
	sender     *mockSender
}

func newComponents() *components {
	machineConfig := smachine.SlotMachineConfig{
		PollingPeriod:     500 * time.Millisecond,
		PollingTruncate:   1 * time.Millisecond,
		SlotPageSize:      1000,
		ScanCountLimit:    100000,
		SlotMachineLogger: statemachine.ConveyorLoggerFactory{},
	}

	conv := conveyor.NewPulseConveyor(context.Background(), conveyor.PulseConveyorConfig{
		ConveyorMachineConfig: machineConfig,
		SlotMachineConfig:     machineConfig,
		EventlessSleep:        100 * time.Millisecond,
		MinCachePulseAge:      100,
		MaxPastPulseAge:       1000,
	}, smachines.CommonFactory, nil)

	worker := smachines.NewWorker()
	worker.AttachTo(conv)

	sender := newMockSender()
	conv.AddDependency(smachines.NewHashingAdapter())
	conv.AddDependency(smachines.NewSyncAdapter())
	conv.PutDependency("insolar.PlatformCryptographyScheme", platformpolicy.NewPlatformCryptographyScheme())
	conv.AddDependency(store.NewRecordStore())
	conv.PutDependency("bus.Sender", sender)

	disp := smachines.NewDispatcher(conv)

	return &components{
		dispatcher: disp,
		sender:     sender,
	}
}

func Benchmark(b *testing.B) {
	ctx := context.Background()
	cmp := newComponents()
	disp := cmp.dispatcher
	sender := cmp.sender

	meta := payload.Meta{
		Payload: payload.MustNewMessage(&payload.V2SetRequestResult{
			ObjectID: gen.ID(),
			SideEffect: &record.Material{
				Virtual: record.Wrap(&record.Amend{}),
			},
		}).Payload,
		Pulse: pulse.MinTimePulse + 10,
	}
	buf, err := meta.Marshal()
	require.NoError(b, err)
	msg := message.NewMessage("", buf)
	disp.BeginPulse(ctx, insolar.Pulse{
		PulseNumber:      pulse.MinTimePulse + 10,
		NextPulseNumber:  pulse.MinTimePulse + 20,
		PrevPulseNumber:  pulse.MinTimePulse + 10,
		EpochPulseNumber: pulse.MinTimePulse + 10,
	})

	for n := 0; n < b.N; n++ {
		sender.init(smachines.RecordBatchThreshold)
		for i := 0; i < smachines.RecordBatchThreshold; i++ {
			_ = disp.Process(msg)
		}
		sender.wait()
	}
}
