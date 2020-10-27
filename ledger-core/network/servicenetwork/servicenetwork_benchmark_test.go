package servicenetwork

import (
	"context"
	"encoding/binary"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/controller"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork"
	"github.com/insolar/assured-ledger/ledger-core/network/transport"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms/legacyhost"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/component-manager"
	"testing"
	"time"
)

const method = "bench"

type BenchSendFunc func(v benchSender, payload []byte)

func BenchmarkThroughput(b *testing.B) {
	n1 := gen.UniqueGlobalRef()
	n2 := gen.UniqueGlobalRef()

	h1, _ := legacyhost.NewHostNS("127.0.0.1:32001", n1, 1)
	h2, _ := legacyhost.NewHostNS("127.0.0.1:32002", n2, 2)

	time.Sleep(time.Second)

	ctx := context.Background()

	resolver := &Resolver{n1: n1, n2: n2, h1: h1, h2: h2}

	ctrl1 := createController(ctx, *h1,
		configuration.Transport{
			Protocol: "TCP",
			Address:  h1.Address.String(),
		},
		resolver,
		func(bytes []byte) {
			// no-op
		})

	ack := make([]byte, 1)
	ack[0] = 1

	results := make(chan []byte, 16)

	var ctrl2 controller.RPCController

	ctrl2 = createController(ctx, *h2,
		configuration.Transport{
			Protocol: "TCP",
			Address:  h2.Address.String(),
		},
		resolver,
		func(payload []byte) {
			results <- payload
			if _, err := ctrl2.SendBytes(ctx, n1, method, ack); err != nil {
				panic(err)
			}
		})

	b.Run("remote", func(b *testing.B) {
		bench := benchSender{
			toAddr:  n2,
			results: results,
			ctl:     ctrl1,
		}

		b.Run("send", func(b *testing.B) {
			bench.runAllSizes(b, func(b *testing.B, payloadSize int) {
				bench.throughput(b, payloadSize, func(v benchSender, payload []byte) {
					if _, err := v.ctl.SendBytes(ctx, v.toAddr, method, payload); err != nil {
						panic(err)
					}
				})
			})
		})
	})
}

func BenchmarkLatency(b *testing.B) {
	n1 := gen.UniqueGlobalRef()
	n2 := gen.UniqueGlobalRef()

	h1, _ := legacyhost.NewHostNS("127.0.0.1:32003", n1, 1)
	h2, _ := legacyhost.NewHostNS("127.0.0.1:32004", n2, 2)

	time.Sleep(time.Second)

	ctx := context.Background()

	resolver := &Resolver{n1: n1, n2: n2, h1: h1, h2: h2}

	ctrl1 := createController(ctx, *h1,
		configuration.Transport{
			Protocol: "TCP",
			Address:  h1.Address.String(),
		},
		resolver,
		func(bytes []byte) {
			// no-op
		})

	ack := make([]byte, 1)
	ack[0] = 1

	results := make(chan []byte, 16)

	var ctrl2 controller.RPCController

	ctrl2 = createController(ctx, *h2,
		configuration.Transport{
			Protocol: "TCP",
			Address:  h2.Address.String(),
		},
		resolver,
		func(payload []byte) {
			results <- payload
			if _, err := ctrl2.SendBytes(ctx, n1, method, ack); err != nil {
				panic(err)
			}
		})

	b.Run("remote", func(b *testing.B) {
		bench := benchSender{
			toAddr:  n2,
			results: results,
			ctl:     ctrl1,
		}

		b.Run("send", func(b *testing.B) {
			bench.runAllSizes(b, func(b *testing.B, payloadSize int) {
				bench.latency(b, payloadSize, func(v benchSender, payload []byte) {
					if _, err := v.ctl.SendBytes(ctx, v.toAddr, method, payload); err != nil {
						panic(err)
					}
				})
			})
		})
	})
}

type benchSender struct {
	toAddr  reference.Global
	results chan []byte
	ctl     controller.RPCController
}

func (v benchSender) runAllSizes(b *testing.B, fn func(b *testing.B, payloadSize int)) {
	b.Run("0.1k", func(b *testing.B) {
		fn(b, 100)
	})

	b.Run("1k", func(b *testing.B) {
		fn(b, 1<<10)
	})

	b.Run("4k", func(b *testing.B) {
		fn(b, 1<<12)
	})

	b.Run("16k", func(b *testing.B) {
		fn(b, 1<<14)
	})

	b.Run("128k", func(b *testing.B) {
		fn(b, 1<<17)
	})

	b.Run("1M", func(b *testing.B) {
		fn(b, 1<<20)
	})
}

func (v benchSender) throughput(b *testing.B, payloadSize int, fn BenchSendFunc) {
	payload := make([]byte, payloadSize)
	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(payload)))

	received := 0
	for i := b.N; i > 0; i-- {
		fn(v, payload)
		select {
		case <-v.results:
			received++
		default:
		}
	}
	for received < b.N {
		<-v.results
		received++
	}
}

func (v benchSender) latency(b *testing.B, payloadSize int, fn BenchSendFunc) {
	payload := make([]byte, payloadSize)

	b.ResetTimer()
	b.ReportAllocs()

	for i := b.N; i > 0; i-- {
		nanos := time.Now().UnixNano()
		binary.LittleEndian.PutUint64(payload, uint64(nanos))
		fn(v, payload)
		<-v.results
	}
}

func createController(
	ctx context.Context,
	host legacyhost.Host,
	tcp configuration.Transport,
	resolver network.RoutingTable,
	receiver func(bytes []byte),
) controller.RPCController {
	f1 := transport.NewFactory(tcp)
	n1, _ := hostnetwork.NewNetwork(host)
	controller := controller.NewRPCController(network.ConfigureOptions(configuration.Configuration{}))

	manager := component.NewManager(nil)

	manager.Inject(controller, resolver, n1, f1)

	err := manager.Start(ctx)
	if err != nil {
		panic(err)
	}

	res := make([]byte, 1)
	res[0] = 1

	controller.RemoteProcedureRegister(method, func(ctx context.Context, args []byte) ([]byte, error) {
		receiver(args)
		return res, nil
	})

	ctr := controller.(component.Initer)
	ctr.Init(ctx)

	return controller
}

type Resolver struct {
	n1 reference.Global
	n2 reference.Global
	h1 *legacyhost.Host
	h2 *legacyhost.Host
}

func (r *Resolver) Resolve(ref reference.Global) (*legacyhost.Host, error) {
	switch ref {
	case r.n1:
		return r.h1, nil
	case r.n2:
		return r.h2, nil
	default:
		panic("Unexpected reference")
	}
}
