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

type testCasesStruct struct {
	testName string
	receiver ReceiverFunc
	sendFunc BenchType
}

type resultChan struct {
	ch chan []byte
}

var result = resultChan{ch: make(chan []byte)}
var h *UnitProtoServersHolder

// WARNING! Benchmark is unstable due to packet drops on overflow.
func BenchmarkThroughput(b *testing.B) {
	// TODO https://insolar.atlassian.net/browse/PLAT-826
	// workaround with set max value for case above
	maxReceiveExceptions = math.MaxInt64
	defer func() {
		maxReceiveExceptions = 1 << 8
	}()

	results := make(chan []byte, 16)

	prf := &UnitProtoServerProfile{
		config: uniserver.ServerConfig{
			BindingAddress: "127.0.0.1:0",
			UDPMaxSize:     0,
			UDPParallelism: 4,
			PeerLimit:      -1,
		},
		desFactory: &TestDeserializationByteFactory{},
	}

	var testCases = []testCasesStruct{
		// {"shipToBody", result.receiverResults, result.shipToBody},
		// {"shipToHead", result.receiverResults, result.shipToHead},
		{"shipToHeadAndBody", result.receiverPullBody, result.shipToHeadAndBody},
	}

	for _, testCase := range testCases {
		h = NewUnitProtoServersHolder(TestLogAdapter{t: b})
		defer h.stop()

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

		bench := benchSender{
			toAddr:  srv1.directAddress(),
			results: results,
			ctl:     srv2.service,
		}

		b.Run(testCase.testName+"/0.1k", func(b *testing.B) {
			bench.throughput(b, 100, testCase.sendFunc)
		})

		b.Run(testCase.testName+"/1k", func(b *testing.B) {
			bench.throughput(b, 1<<10, testCase.sendFunc)
		})

		b.Run(testCase.testName+"/4k", func(b *testing.B) {
			bench.throughput(b, 1<<12, testCase.sendFunc)
		})

		if testCase.testName != "shipToHead" {

			b.Run(testCase.testName+"/16k", func(b *testing.B) {
				bench.throughput(b, 1<<14, testCase.sendFunc)
			})

			b.Run(testCase.testName+"/128k", func(b *testing.B) {
				bench.throughput(b, 1<<17, testCase.sendFunc)
			})

			b.Run(testCase.testName+"/1M", func(b *testing.B) {
				bench.throughput(b, 1<<20, testCase.sendFunc)
			})

			b.Run(testCase.testName+"/8M", func(b *testing.B) {
				bench.throughput(b, 1<<23, testCase.sendFunc)
			})
		}
	}
	// b.Run("loopback", func(b *testing.B) {
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
	// })
}

// WARNING! Benchmark is unstable due to packet drops on overflow.
func BenchmarkLatency(b *testing.B) {
	// TODO https://insolar.atlassian.net/browse/PLAT-826
	// workaround with set max value for case above
	maxReceiveExceptions = math.MaxInt64
	defer func() {
		maxReceiveExceptions = 1 << 8
	}()

	results := make(chan []byte, 1)

	prf := &UnitProtoServerProfile{
		config: uniserver.ServerConfig{
			BindingAddress: "127.0.0.1:0",
			UDPMaxSize:     0,
			UDPParallelism: 4,
			PeerLimit:      -1,
		},
		desFactory: &TestDeserializationByteFactory{},
	}

	var testCases = []testCasesStruct{
		{"shipToBody", result.receiverEmpty, result.shipToBody},
		{"shipToHead", result.receiverEmpty, result.shipToHead},
		{"shipToHeadAndBody", result.receiverPullBody, result.shipToHeadAndBody},
	}

	for _, testCase := range testCases {
		h := NewUnitProtoServersHolder(TestLogAdapter{t: b})
		defer h.stop()

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

		bench := benchSender{
			toAddr:  srv1.directAddress(),
			results: results,
			ctl:     srv2.service,
		}

		b.Run(testCase.testName+"/0.1k", func(b *testing.B) {
			bench.latency(b, 100, testCase.sendFunc)
		})

		b.Run(testCase.testName+"/4k", func(b *testing.B) {
			bench.latency(b, 1<<12, testCase.sendFunc)
		})
		if testCase.testName != "shipToHead" {

			b.Run(testCase.testName+"/128k", func(b *testing.B) {
				bench.latency(b, 1<<17, testCase.sendFunc)
			})

			b.Run(testCase.testName+"/1M", func(b *testing.B) {
				bench.latency(b, 1<<20, testCase.sendFunc)
			})

			b.Run(testCase.testName+"/8M", func(b *testing.B) {
				bench.latency(b, 1<<23, testCase.sendFunc)
			})
		}
	}
	// b.Run("loopback", func(b *testing.B) {
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
	// })
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
			// println(received, b.N, " in-loop")
		default:
		}
	}
	for received < b.N {
		<-v.results
		received++
		// println(received, b.N, " off-loop")
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

func (r *resultChan) shipToBody(v benchSender, payload []byte) {
	err := v.ctl.ShipTo(v.toAddr, Shipment{Body: &TestBytes{payload}})
	if err != nil {
		panic(err)
	}
}

func (r *resultChan) shipToHead(v benchSender, payload []byte) {
	err := v.ctl.ShipTo(v.toAddr, Shipment{Head: &TestBytes{payload}})
	if err != nil {
		panic(err)
	}
}

func (s *resultChan) shipToHeadAndBody(v benchSender, payload []byte) {
	head := TestString{string(payload[:64])}
	body := TestString{string(payload)}

	err := v.ctl.ShipTo(v.toAddr, Shipment{
		Head: &head,
		Body: &body,
	})
	if err != nil {
		panic(err)
	}

	<-s.ch
}

func (r *resultChan) receiverPullBody(a ReturnAddress, done nwapi.PayloadCompleteness, _ interface{}) error {
	err := h.server(1).service.PullBody(a, ShipmentRequest{
		ReceiveFn: func(add ReturnAddress, done nwapi.PayloadCompleteness, body interface{}) error {
			if !done {
				panic("")
			}

			r.ch <- []byte{}

			return nil
		},
	})

	if err != nil {
		panic(err)
	}
	return nil
}

func (r *resultChan) receiverResults(a ReturnAddress, _ nwapi.PayloadCompleteness, v interface{}) error {
	r.ch <- v.(*TestBytes).S
	return nil
}
func (r *resultChan) receiverEmpty(_ ReturnAddress, _ nwapi.PayloadCompleteness, _ interface{}) error {
	r.ch <- nil
	return nil
}
