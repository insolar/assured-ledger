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

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l2/uniserver"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

type SendFuncType func(v benchSender, payload []byte)

type testCasesStruct struct {
	testName string
	receiver ReceiverFunc
	sendFunc SendFuncType
}

type initServerData struct {
	serverConf uniserver.ServerConfig
	receiver   ReceiverFunc
}

type benchSender struct {
	toAddr DeliveryAddress
	ctl2   Service
}

var result = resultChan{ch: make(chan []byte)}

// WARNING! Benchmark is unstable due to packet drops on overflow.
func BenchmarkThroughput(b *testing.B) {
	// TODO https://insolar.atlassian.net/browse/PLAT-826
	// workaround with set max value for case above
	maxReceiveExceptions = math.MaxInt64
	results := make(chan []byte, 16)

	srv2 := initServerData{
		serverConf: uniserver.ServerConfig{
			BindingAddress: "127.0.0.1:0",
			UDPMaxSize:     0,
			UDPParallelism: 4,
			PeerLimit:      -1,
		},
		receiver: noopReceiver,
	}
	srv1 := srv2

	var testCases = []testCasesStruct{
		{"shipToBody", result.receiverResults, result.shipToBody},
		{"shipToHead", result.receiverResults, result.shipToHead},
		{"shipToHeadAndBody", result.receiverPullBody, result.shipToHeadAndBody},
	}

	for _, testCase := range testCases {
		println(testCase.testName)

		srv1.receiver = testCase.receiver

		sender, stopFn := createPipe(b, srv1, srv2)
		defer stopFn()

		result.ch = results
		bench := sender

		b.Run("0.1k", func(b *testing.B) {
			bench.throughput(b, 100, testCase.sendFunc)
		})

		b.Run("1k", func(b *testing.B) {
			bench.throughput(b, 1<<10, testCase.sendFunc)
		})

		b.Run("4k", func(b *testing.B) {
			bench.throughput(b, 1<<12, testCase.sendFunc)
		})

		if testCase.testName != "shipToHead" {

			b.Run("16k", func(b *testing.B) {
				bench.throughput(b, 1<<14, testCase.sendFunc)
			})

			b.Run("128k", func(b *testing.B) {
				bench.throughput(b, 1<<17, testCase.sendFunc)
			})

			b.Run("1M", func(b *testing.B) {
				bench.throughput(b, 1<<20, testCase.sendFunc)
			})

			b.Run("8M", func(b *testing.B) {
				bench.throughput(b, 1<<23, testCase.sendFunc)
			})
		}
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
	// TODO https://insolar.atlassian.net/browse/PLAT-826
	// workaround with set max value for case above
	maxReceiveExceptions = math.MaxInt64
	results := make(chan []byte, 1)
	srv2 := initServerData{
		serverConf: uniserver.ServerConfig{
			BindingAddress: "127.0.0.1:0",
			UDPMaxSize:     0,
			UDPParallelism: 4,
			PeerLimit:      -1,
		},
		receiver: noopReceiver,
	}
	srv1 := srv2

	var testCases = []testCasesStruct{
		{"shipToBody", result.receiverEmpty, result.shipToBody},
		{"shipToHead", result.receiverEmpty, result.shipToHead},
		{"shipToHeadAndBody", result.receiverPullBody, result.shipToHeadAndBody},
	}

	for _, testCase := range testCases {
		println(testCase.testName)

		srv1.receiver = testCase.receiver

		sender, stopFn := createPipe(b, srv1, srv2)
		defer stopFn()

		result.ch = results
		bench := sender

		b.Run("0.1k", func(b *testing.B) {
			bench.latency(b, 100, testCase.sendFunc)
		})

		b.Run("4k", func(b *testing.B) {
			bench.latency(b, 1<<12, testCase.sendFunc)
		})

		b.Run("128k", func(b *testing.B) {
			bench.latency(b, 1<<17, testCase.sendFunc)
		})

		b.Run("1M", func(b *testing.B) {
			bench.latency(b, 1<<20, testCase.sendFunc)
		})

		b.Run("8M", func(b *testing.B) {
			bench.latency(b, 1<<23, testCase.sendFunc)
		})
	}
}

func createPipe(t testing.TB, server1, server2 initServerData) (benchSender, func()) {
	servers = make([]*UniprotoServer, 0)
	var idWithPortFn func(nwapi.Address) bool
	if server1.serverConf.BindingAddress == server2.serverConf.BindingAddress {
		addr := nwapi.NewHostPort(server1.serverConf.BindingAddress, true)
		if !addr.IsLoopback() {
			addr = addr.HostIdentity()
			idWithPortFn = func(address nwapi.Address) bool {
				return address.HostIdentity() == addr
			}
		}
	}

	srv1 := createService(t, server1.receiver, server1.serverConf, idWithPortFn)
	srv2 := createService(t, server2.receiver, server2.serverConf, idWithPortFn)

	return benchSender{
			toAddr: NewDirectAddress(1),
			ctl2:   srv2.service,
		}, func() {
			srv1.disp.Stop()
			srv2.disp.Stop()
		}
}

func (v benchSender) throughput(b *testing.B, payloadSize int, funcName SendFuncType) {
	payload := make([]byte, payloadSize)
	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(payload)))

	received := 100
	for i := b.N; i > 0; i-- {
		funcName(v, payload)
		select {
		case <-result.ch:
			received++
			// println(received, b.N, " in-loop")
		default:
		}
	}
	for received < b.N {
		<-result.ch
		received++
		// println(received, b.N, " off-loop")
	}
}

func (v benchSender) latency(b *testing.B, payloadSize int, funcName SendFuncType) {
	payload := make([]byte, payloadSize)
	b.SetBytes(int64(len(payload)))

	b.ResetTimer()
	b.ReportAllocs()

	for i := b.N; i > 0; i-- {
		nanos := time.Now().UnixNano()
		binary.LittleEndian.PutUint64(payload, uint64(nanos))
		funcName(v, payload)
		<-result.ch
	}
}

func (r *resultChan) shipToBody(v benchSender, payload []byte) {
	err := v.ctl2.ShipTo(v.toAddr, Shipment{Body: &TestBytes{payload}})
	if err != nil {
		panic(err)
	}
}

func (r *resultChan) shipToHead(v benchSender, payload []byte) {
	err := v.ctl2.ShipTo(v.toAddr, Shipment{Head: &TestBytes{payload}})
	if err != nil {
		panic(err)
	}
}

type resultChan struct {
	ch chan []byte
}

func (s *resultChan) shipToHeadAndBody(v benchSender, payload []byte) {
	head := TestString{string(payload[:64])}
	body := TestString{string(payload)}

	err := v.ctl2.ShipTo(v.toAddr, Shipment{
		Head: &head,
		Body: &body,
	})
	if err != nil {
		panic(err)
	}

	<-s.ch
}

func (r *resultChan) receiverPullBody(a ReturnAddress, done nwapi.PayloadCompleteness, _ interface{}) error {
	srv1 := getServerByIndex(1)

	err := srv1.service.PullBody(a, ShipmentRequest{
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
