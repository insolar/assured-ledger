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

type BenchSendFunc func(v benchSender, payload []byte)

// WARNING! Benchmark is unstable due to packet drops on overflow.
func BenchmarkThroughput(b *testing.B) {
	// TODO https://insolar.atlassian.net/browse/PLAT-826
	// workaround with set max value for case above
	maxReceiveExceptions = math.MaxInt64
	defer func() {
		maxReceiveExceptions = 1 << 8
	}()

	results := make(chan []byte, 16)

	h := NewUnitProtoServersHolder(TestLogAdapter{t: b})
	defer h.stop()

	prf := &UnitProtoServerProfile{
		config: uniserver.ServerConfig{
			BindingAddress: "127.0.0.1:0",
			UDPMaxSize:     0,
			UDPParallelism: 4,
			PeerLimit:      -1,
		},
		desFactory: &TestDeserializationByteFactory{},
	}

	var srv1 *UnitProtoServer
	var err error
	srv1, err = h.createServiceWithProfile(prf, func(a ReturnAddress, done nwapi.PayloadCompleteness, val interface{}) error {
		if !done {
			err := srv1.service.PullBody(a, ShipmentRequest{
				ReceiveFn: func(a ReturnAddress, _ nwapi.PayloadCompleteness, val interface{}) error {
					bb := val.(*TestBytes).S
					results <- bb
					return nil
				},
			})
			require.NoError(b, err)
			return nil
		}

		bb := val.(*TestBytes).S
		results <- bb
		return nil
	})
	require.NoError(b, err)

	var srv2 *UnitProtoServer
	srv2, err = h.createServiceWithProfile(prf, func(a ReturnAddress, done nwapi.PayloadCompleteness, val interface{}) error {
		if !done {
			err := srv2.service.PullBody(a, ShipmentRequest{
				ReceiveFn: func(a ReturnAddress, _ nwapi.PayloadCompleteness, val interface{}) error {
					bb := val.(*TestBytes).S
					results <- bb
					return nil
				},
			})
			require.NoError(b, err)
			return nil
		}

		bb := val.(*TestBytes).S
		results <- bb
		return nil
	})
	require.NoError(b, err)

	b.Run("remote", func(b *testing.B) {
		bench := benchSender{
			toAddr:  srv1.directAddress(),
			results: results,
			ctl:     srv2.service,
		}

		// b.Run("head+body", func(b *testing.B) {
		// 	bench.runAllSizes(b, func(b *testing.B, payloadSize int) {
		// 		bench.throughput(b, payloadSize, shipToHead)
		// 	})
		// })

		b.Run("body", func(b *testing.B) {
			bench.runAllSizes(b, func(b *testing.B, payloadSize int) {
				bench.throughput(b, payloadSize, shipToBody)
			})
		})
	})
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

	h := NewUnitProtoServersHolder(TestLogAdapter{t: b})
	defer h.stop()

	prf := &UnitProtoServerProfile{
		config: uniserver.ServerConfig{
			BindingAddress: "127.0.0.1:0",
			UDPMaxSize:     0,
			UDPParallelism: 4,
			PeerLimit:      -1,
		},
		desFactory: &TestDeserializationByteFactory{},
	}

	var srv1 *UnitProtoServer
	var err error
	srv1, err = h.createServiceWithProfile(prf, func(a ReturnAddress, done nwapi.PayloadCompleteness, val interface{}) error {
		if !done {
			err := srv1.service.PullBody(a, ShipmentRequest{
				ReceiveFn: func(a ReturnAddress, _ nwapi.PayloadCompleteness, val interface{}) error {
					results <- nil
					return nil
				},
			})
			require.NoError(b, err)
			return nil
		}

		results <- nil
		return nil
	})
	require.NoError(b, err)

	var srv2 *UnitProtoServer
	srv2, err = h.createServiceWithProfile(prf, func(a ReturnAddress, done nwapi.PayloadCompleteness, val interface{}) error {
		if !done {
			err := srv2.service.PullBody(a, ShipmentRequest{
				ReceiveFn: func(a ReturnAddress, _ nwapi.PayloadCompleteness, val interface{}) error {
					results <- nil
					return nil
				},
			})
			require.NoError(b, err)
			return nil
		}

		results <- nil
		return nil
	})
	require.NoError(b, err)

	b.Run("remote", func(b *testing.B) {
		bench := benchSender{
			toAddr:  srv1.directAddress(),
			results: results,
			ctl:     srv2.service,
		}

		// b.Run("head+body", func(b *testing.B) {
		// 	bench.runAllSizes(b, func(b *testing.B, payloadSize int) {
		// 		bench.throughput(b, payloadSize, shipToHead)
		// 	})
		// })

		b.Run("body", func(b *testing.B) {
			bench.runAllSizes(b, func(b *testing.B, payloadSize int) {
				bench.latency(b, payloadSize, shipToBody)
			})
		})
	})

	b.Run("loopback", func(b *testing.B) {
		bench := benchSender{
			toAddr:  srv2.directAddress(), // loopback
			results: results,
			ctl:     srv2.service,
		}

		// b.Run("head+body", func(b *testing.B) {
		// 	bench.runAllSizes(b, func(b *testing.B, payloadSize int) {
		// 		bench.throughput(b, payloadSize, shipToHead)
		// 	})
		// })

		b.Run("body", func(b *testing.B) {
			bench.runAllSizes(b, func(b *testing.B, payloadSize int) {
				bench.latency(b, payloadSize, shipToBody)
			})
		})
	})
}

type benchSender struct {
	toAddr  DeliveryAddress
	results chan []byte
	ctl     Service
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

func shipToBody(v benchSender, payload []byte) {
	err := v.ctl.ShipTo(v.toAddr, Shipment{Body: &TestBytes{payload}})
	if err != nil {
		panic(err)
	}
}

func shipToHead(v benchSender, payload []byte) {
	err := v.ctl.ShipTo(v.toAddr, Shipment{Head: &TestBytes{payload[:64]}, Body: &TestBytes{payload}})
	if err != nil {
		panic(err)
	}
}
