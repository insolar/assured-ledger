// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"encoding/binary"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l2/uniserver"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

type BenchType func(v benchSender, payload []byte)

// WARNING! Benchmark is unstable due to packet drops on overflow.
func BenchmarkThroughput(b *testing.B) {
	//TODO https://insolar.atlassian.net/browse/PLAT-826
	// workaround with set max value for case above
	maxReceiveExceptions = math.MaxInt64
	defer func() {
		maxReceiveExceptions = 1 << 8
	}()

	results := make(chan []byte, 16)

	h := NewUnitProtoServersHolder(TestLogAdapter{t: b})
	defer h.stop()

	prf := &UnitProtoServerProfile{
		config: &uniserver.ServerConfig{
			BindingAddress: "127.0.0.1:0",
			UDPMaxSize:     0,
			UDPParallelism: 4,
			PeerLimit:      -1,
		},
		desFactory: &TestDeserializationByteFactory{},
	}

	srv1, err := h.createServiceWithProfile(prf, func(a ReturnAddress, done nwapi.PayloadCompleteness, val interface{}) error {
		bb := val.(*TestBytes).S
		results <- bb
		return nil
	})
	require.NoError(b, err)

	srv2, err := h.createServiceWithProfile(prf, func(a ReturnAddress, done nwapi.PayloadCompleteness, val interface{}) error {
		bb := val.(*TestBytes).S
		results <- bb
		return nil
	})
	require.NoError(b, err)

	testCases := []BenchType{
		shipToBody,
		shipToHead,
	}

	for _, runFunc := range testCases {
		b.Run("remote", func(b *testing.B) {
			bench := benchSender{
				toAddr:  srv1.directAddress(),
				results: results,
				ctl:     srv2.service,
			}

			b.Run("0.1k", func(b *testing.B) {
				bench.throughput(b, 100, runFunc)
			})

			b.Run("1k", func(b *testing.B) {
				bench.throughput(b, 1<<10, runFunc)
			})

			b.Run("4k", func(b *testing.B) {
				bench.throughput(b, 1<<12, runFunc)
			})

			b.Run("16k", func(b *testing.B) {
				bench.throughput(b, 1<<14, runFunc)
			})

			b.Run("128k", func(b *testing.B) {
				bench.throughput(b, 1<<17, runFunc)
			})

			b.Run("1M", func(b *testing.B) {
				bench.throughput(b, 1<<20, runFunc)
			})
		})

		//b.Run("loopback", func(b *testing.B) {
		//	bench := benchSender{
		//		toAddr:  srv2.directAddress(), // loopback
		//		results: results,
		//		ctl:     srv2.service,
		//	}
		//
		//	b.Run("0.1k", func(b *testing.B) {
		//		bench.throughput(b, 100, runFunc)
		//	})
		//
		//	b.Run("1k", func(b *testing.B) {
		//		bench.throughput(b, 1<<10, runFunc)
		//	})
		//
		//	b.Run("4k", func(b *testing.B) {
		//		bench.throughput(b, 1<<12, runFunc)
		//	})
		//
		//	b.Run("16k", func(b *testing.B) {
		//		bench.throughput(b, 1<<14, runFunc)
		//	})
		//
		//	b.Run("128k", func(b *testing.B) {
		//		bench.throughput(b, 1<<17, runFunc)
		//	})
		//
		//	b.Run("1M", func(b *testing.B) {
		//		bench.throughput(b, 1<<20, runFunc)
		//	})
		//})
	}
}

// WARNING! Benchmark is unstable due to packet drops on overflow.
func BenchmarkLatency(b *testing.B) {
	//TODO https://insolar.atlassian.net/browse/PLAT-826
	// workaround with set max value for case above
	maxReceiveExceptions = math.MaxInt64
	defer func() {
		maxReceiveExceptions = 1 << 8
	}()

	results := make(chan []byte, 1)

	h := NewUnitProtoServersHolder(TestLogAdapter{t: b})
	defer h.stop()

	prf := &UnitProtoServerProfile{
		config: &uniserver.ServerConfig{
			BindingAddress: "127.0.0.1:0",
			UDPMaxSize:     0,
			UDPParallelism: 4,
			PeerLimit:      -1,
		},
		desFactory: &TestDeserializationByteFactory{},
	}

	srv1, err := h.createServiceWithProfile(prf, func(a ReturnAddress, done nwapi.PayloadCompleteness, val interface{}) error {
		results <- nil
		return nil
	})
	require.NoError(b, err)

	srv2, err := h.createServiceWithProfile(prf, func(a ReturnAddress, done nwapi.PayloadCompleteness, val interface{}) error {
		results <- nil
		return nil
	})
	require.NoError(b, err)

	testCases := []BenchType{
		shipToBody,
		shipToHead,
	}

	for _, runFunc := range testCases {
		b.Run("remote", func(b *testing.B) {
			bench := benchSender{
				toAddr:  srv1.directAddress(),
				results: results,
				ctl:     srv2.service,
			}

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
		})

		//b.Run("loopback", func(b *testing.B) {
		//	bench := benchSender{
		//		toAddr:  srv2.directAddress(),
		//		results: results,
		//		ctl:     srv2.service,
		//	}
		//
		//	b.Run("0.1k", func(b *testing.B) {
		//		bench.latency(b, 100, runFunc)
		//	})
		//
		//	b.Run("4k", func(b *testing.B) {
		//		bench.latency(b, 1<<12, runFunc)
		//	})
		//
		//	b.Run("128k", func(b *testing.B) {
		//		bench.latency(b, 1<<17, runFunc)
		//	})
		//
		//	b.Run("1M", func(b *testing.B) {
		//		bench.latency(b, 1<<20, runFunc)
		//	})
		//
		//	b.Run("8M", func(b *testing.B) {
		//		bench.latency(b, 1<<23, runFunc)
		//	})
		//
		//	b.Run("32M", func(b *testing.B) {
		//		bench.latency(b, 1<<25, runFunc)
		//	})
		//
		//	b.Run("64M", func(b *testing.B) {
		//		bench.latency(b, 1<<26, runFunc)
		//	})
		//
		//	b.Run("128M", func(b *testing.B) {
		//		bench.latency(b, 1<<27, runFunc)
		//	})
		//})
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

////////////////////////////

func shipToBody(v benchSender, payload []byte) {
	err := v.ctl.ShipTo(v.toAddr, Shipment{Body: &TestBytes{payload}})
	if err != nil {
		panic(err)
	}
}

func shipToHead(v benchSender, payload []byte) {
	err := v.ctl.ShipTo(v.toAddr, Shipment{Head: &TestBytes{payload[:64]}})
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
