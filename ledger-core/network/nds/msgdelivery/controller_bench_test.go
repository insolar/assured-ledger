// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l2/uniserver"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

type BenchType func(v benchSender, payload []byte)

// WARNING! Benchmark is unstable due to packet drops on overflow.
func BenchmarkThroughput(b *testing.B) {
	results := make(chan []byte, 16)
	recv := func(a ReturnAddress, _ nwapi.PayloadCompleteness, v interface{}) error {
		results <- nil
		return nil
	}

	sender, stopFn := createPipe(b, "127.0.0.1:0", "127.0.0.1:0", 0, recv)
	defer stopFn()

	sender.results = results

	testCases := []BenchType{
		shipToBody,
		shipToHead,
	}

	for _, runFunc := range testCases {
		bench := sender

		b.Run("0.1k", func(b *testing.B) {
			bench.throughput(b, 100, runFunc)
		})

		// b.Run("1k", func(b *testing.B) {
		// 	bench.throughput(b, 1<<10, runFunc)
		// })
		//
		// b.Run("4k", func(b *testing.B) {
		// 	bench.throughput(b, 1<<12, runFunc)
		// })
		//
		// b.Run("16k", func(b *testing.B) {
		// 	bench.throughput(b, 1<<14, runFunc)
		// })
		//
		// b.Run("128k", func(b *testing.B) {
		// 	bench.throughput(b, 1<<17, runFunc)
		// })
		//
		// // head -
		// b.Run("1M", func(b *testing.B) {
		// 	bench.throughput(b, 1<<20, runFunc)
		// })
		//
		// b.Run("8M", func(b *testing.B) {
		// 	bench.throughput(b, 1<<23, runFunc)
		// })
		//
		// b.Run("32M", func(b *testing.B) {
		// 	bench.throughput(b, 1<<25, runFunc)
		// })
		//
		// b.Run("64M", func(b *testing.B) {
		// 	bench.throughput(b, 1<<26, runFunc)
		// })
		//
		// b.Run("128M", func(b *testing.B) {
		// 	bench.throughput(b, 1<<27, runFunc)
		// })
	}

	// b.Run("loopback", func(b *testing.B) {
	// 	bench := sender
	// 	bench.toAddr = NewDirectAddress(2) // loopback
	//
	// 	b.Run("0.1k", func(b *testing.B) {
	// 		bench.throughput(b, 100)
	// 	})
	//
	// 	b.Run("4k", func(b *testing.B) {
	// 		bench.throughput(b, 1<<12)
	// 	})
	//
	// 	b.Run("16k", func(b *testing.B) {
	// 		bench.throughput(b, 1<<14)
	// 	})
	// })
}

// WARNING! Benchmark is unstable due to packet drops on overflow.
func BenchmarkLatency(b *testing.B) {
	results := make(chan []byte, 1)
	recv := func(a ReturnAddress, _ nwapi.PayloadCompleteness, v interface{}) error {
		results <- nil
		return nil
	}
	sender, stopFn := createPipe(b, "127.0.0.1:0", "127.0.0.1:0", 0, recv)
	defer stopFn()

	sender.results = results

	testCases := []BenchType{
		shipToBody,
		shipToHead,
	}

	for _, runFunc := range testCases {
		bench := sender

		b.Run("0.1k", func(b *testing.B) {
			bench.latency(b, 100, runFunc)
		})

		b.Run("4k", func(b *testing.B) {
			bench.latency(b, 1<<12, runFunc)
		})

		b.Run("128k", func(b *testing.B) {
			bench.latency(b, 1<<17, runFunc)
		})

		b.Run("1M", func(b *testing.B) {
			bench.latency(b, 1<<20, runFunc)
		})

		b.Run("8M", func(b *testing.B) {
			bench.latency(b, 1<<23, runFunc)
		})

		b.Run("32M", func(b *testing.B) {
			bench.latency(b, 1<<25, runFunc)
		})

		b.Run("64M", func(b *testing.B) {
			bench.latency(b, 1<<26, runFunc)
		})

		b.Run("128M", func(b *testing.B) {
			bench.latency(b, 1<<27, runFunc)
		})
	}
}

func createPipe(
	t testing.TB,
	server1, server2 string,
	udpMaxSize int,
	f func(a ReturnAddress, _ nwapi.PayloadCompleteness, v interface{}) error,
) (benchSender, func()) {
	var idWithPortFn func(nwapi.Address) bool
	if server1 == server2 {
		addr := nwapi.NewHostPort(server1, true)
		if !addr.IsLoopback() {
			addr = addr.HostIdentity()
			idWithPortFn = func(address nwapi.Address) bool {
				return address.HostIdentity() == addr
			}
		}
	}

	config := uniserver.ServerConfig{
		BindingAddress: server1,
		UDPMaxSize:     udpMaxSize,
		UDPParallelism: 4,
		PeerLimit:      -1,
	}

	srv1 := createService(t, f, config, idWithPortFn)

	config.BindingAddress = server2
	srv2 := createService(t, f, config, idWithPortFn)

	return benchSender{
			toAddr: NewDirectAddress(1),
			ctl:    srv2.service,
		}, func() {
			srv1.disp.Stop()
			srv2.disp.Stop()
		}
}

type benchSender struct {
	toAddr  DeliveryAddress
	results chan []byte
	ctl     Service
}

func (v benchSender) throughput(b *testing.B, payloadSize int, funcName BenchType) {
	payload := make([]byte, payloadSize)
	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(payload)))

	received := 100
	for i := b.N; i > 0; i-- {
		funcName(v, payload)
		select {
		case <-v.results:
			received++
			//println(received, b.N, " in-loop")
		default:
		}
	}
	for received < b.N {
		<-v.results
		received++
		//println(received, b.N, " off-loop")
	}
}

func (v benchSender) latency(b *testing.B, payloadSize int, funcName BenchType) {
	payload := make([]byte, payloadSize)
	b.SetBytes(int64(len(payload)))

	b.ResetTimer()
	b.ReportAllocs()

	for i := b.N; i > 0; i-- {
		nanos := time.Now().UnixNano()
		binary.LittleEndian.PutUint64(payload, uint64(nanos))
		funcName(v, payload)
		<-v.results
	}
}

// //////////////////////////

func shipToBody(v benchSender, payload []byte) {
	err := v.ctl.ShipTo(v.toAddr, Shipment{Body: &TestBytes{payload}})
	if err != nil {
		panic(err)
	}
}

func shipToHead(v benchSender, payload []byte) {
	err := v.ctl.ShipTo(v.toAddr, Shipment{Head: &TestBytes{payload}})
	if err != nil {
		panic(err)
	}
}

func shipToHeadAndPullBody(v benchSender, payload []byte) {
	head := TestString{string(payload[:64])}
	body := TestString{string(payload)}

	// recv1 := func(a ReturnAddress, done nwapi.PayloadCompleteness, _ interface{}) error {
	// 	err := v.ctl.PullBody(a, ShipmentRequest{
	// 		ReceiveFn: func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
	// 			return nil
	// 		},
	// 	})
	//
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	return nil
	// }
	// todo receiver need before server init

	err := v.ctl.ShipTo(v.toAddr, Shipment{
		Head: &head,
		Body: &body,
	})
	if err != nil {
		panic(err)
	}
}
