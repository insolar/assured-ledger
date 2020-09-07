// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l2/uniserver"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

// WARNING! Benchmark is unstable due to packet drops on overflow.
func BenchmarkThroughput(b *testing.B) {
	results := make(chan []byte, 16)
	sender, stopFn := createPipe(b, "127.0.0.1:0", "127.0.0.1:0", 0, func(bb []byte) {
		results <- bb
	})
	defer stopFn()

	sender.results = results

	b.Run("localhost", func(b *testing.B) {
		bench := sender

		b.Run("0.1k", func(b *testing.B) {
			bench.throughput(b, 100)
		})

		b.Run("1k", func(b *testing.B) {
			bench.throughput(b, 1<<10)
		})

		b.Run("4k", func(b *testing.B) {
			bench.throughput(b, 1<<12)
		})

		b.Run("16k", func(b *testing.B) {
			bench.throughput(b, 1<<14)
		})

		b.Run("128k", func(b *testing.B) {
			bench.throughput(b, 1<<17)
		})

		// b.Run("1M", func(b *testing.B) {
		// 	bench.bench(b, 1<<20)
		// })
	})

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
	sender, stopFn := createPipe(b, "127.0.0.1:0", "127.0.0.1:0", 0, func(bb []byte) {
		// nanos := time.Now().UnixNano()
		// nanos -= int64(binary.LittleEndian.Uint64(bb))
		results <- nil
	})
	defer stopFn()

	sender.results = results

	oob := false

	b.Run("regular", func(b *testing.B) {
		bench := sender

		b.Run("0.1k", func(b *testing.B) {
			bench.latency(b, 100, oob)
		})

		b.Run("4k", func(b *testing.B) {
			bench.latency(b, 1<<12, oob)
		})

		b.Run("128k", func(b *testing.B) {
			bench.latency(b, 1<<17, oob)
		})
	})

	oob = true
	b.Run("asap", func(b *testing.B) {
		bench := sender

		b.Run("0.1k", func(b *testing.B) {
			bench.latency(b, 100, oob)
		})

		b.Run("4k", func(b *testing.B) {
			bench.latency(b, 1<<12, oob)
		})

		b.Run("128k", func(b *testing.B) {
			bench.latency(b, 1<<17, oob)
		})
	})
}

func createPipe(t testing.TB, server1, server2 string, udpMaxSize int, resultFn func([]byte)) (benchSender, func ()) {

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

	vf := TestVerifierFactory{}
	skBytes := [testDigestSize]byte{}
	sk1 := cryptkit.NewSigningKey(longbits.CopyBytes(skBytes[:]), testSigningMethod, cryptkit.PublicAsymmetricKey)
	skBytes[0] = 1
	sk2 := cryptkit.NewSigningKey(longbits.CopyBytes(skBytes[:]), testSigningMethod, cryptkit.PublicAsymmetricKey)

	controller1 := NewController(Protocol, TestDeserializationByteFactory{},
		func(a ReturnAddress, _ nwapi.PayloadCompleteness, v interface{}) error {
			resultFn(v.(*TestBytes).S)
			return nil
		}, nil, TestLogAdapter{t})

	var dispatcher1 uniserver.Dispatcher
	controller1.RegisterWith(dispatcher1.RegisterProtocol)

	ups1 := uniserver.NewUnifiedServer(&dispatcher1, TestLogAdapter{t})
	ups1.SetConfig(uniserver.ServerConfig{
		BindingAddress: server1,
		UDPMaxSize:     udpMaxSize,
		UDPParallelism: 4,
		PeerLimit:      -1,
	})
	ups1.SetIdentityClassifier(idWithPortFn)

	ups1.SetPeerFactory(func(peer *uniserver.Peer) (remapTo nwapi.Address, err error) {
		peer.SetSignatureKey(sk2)
		peer.SetNodeID(2)
		return nwapi.NewHostID(2), nil
	})
	ups1.SetSignatureFactory(vf)

	ups1.StartListen()
	dispatcher1.SetMode(uniproto.AllowAll)

	pm1 := ups1.PeerManager()
	_, err := pm1.AddHostID(pm1.Local().GetPrimary(), 1)
	require.NoError(t, err)

	pr := pulse.NewOnePulseRange(pulse.NewFirstPulsarData(5, longbits.Bits256{}))
	dispatcher1.NextPulse(pr)

	/********************************/

	controller2 := NewController(Protocol, TestDeserializationByteFactory{},
		func(a ReturnAddress, _ nwapi.PayloadCompleteness, v interface{}) error {
			resultFn(v.(*TestBytes).S)
			return nil
		}, nil, TestLogAdapter{t})

	var dispatcher2 uniserver.Dispatcher
	dispatcher2.SetMode(uniproto.NewConnectionMode(0, Protocol))
	controller2.RegisterWith(dispatcher2.RegisterProtocol)
	dispatcher2.Seal()

	ups2 := uniserver.NewUnifiedServer(&dispatcher2, TestLogAdapter{t})
	ups2.SetConfig(uniserver.ServerConfig{
		BindingAddress: server2,
		UDPMaxSize:     udpMaxSize,
		UDPParallelism: 4,
		PeerLimit:      -1,
	})
	ups2.SetIdentityClassifier(idWithPortFn)

	ups2.SetPeerFactory(func(peer *uniserver.Peer) (remapTo nwapi.Address, err error) {
		peer.SetSignatureKey(sk1)
		peer.SetNodeID(1)
		return nwapi.NewHostID(1), nil
	})
	ups2.SetSignatureFactory(vf)

	ups2.StartListen()
	dispatcher2.NextPulse(pr)

	pm2 := ups2.PeerManager()
	_, err = pm2.AddHostID(pm2.Local().GetPrimary(), 2)
	require.NoError(t, err)

	conn21, err := pm2.Manager().ConnectPeer(pm1.Local().GetPrimary())
	require.NoError(t, err)
	require.NoError(t, conn21.Transport().EnsureConnect())

	ctl2 := controller2.NewFacade()

	return benchSender{
			toAddr:  NewDirectAddress(1),
			ctl:     ctl2,
		}, func() {
			dispatcher1.Stop()
			dispatcher2.Stop()
		}
}

type benchSender struct {
	toAddr  DeliveryAddress
	results chan []byte
	ctl     Service
}

func (v benchSender) throughput(b *testing.B, payloadSize int) {
	payload := make([]byte, payloadSize)
	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(payload)))

	received := 100
	for i := b.N; i > 0; i-- {
		err := v.ctl.ShipTo(v.toAddr, Shipment{Body: &TestBytes{payload}})
		if err != nil {
			panic(err)
		}
		select {
		case <- v.results:
			received++
			// println(received, b.N, " in-loop")
		default:
		}
	}
	for ;received < b.N; {
		<- v.results
		received++
		// println(received, b.N, " off-loop")
	}
}

func (v benchSender) latency(b *testing.B, payloadSize int, oob bool) {
	payload := make([]byte, payloadSize)
	b.SetBytes(int64(len(payload)))

	b.ResetTimer()
	b.ReportAllocs()

	for i := b.N; i > 0; i-- {
		nanos := time.Now().UnixNano()
		binary.LittleEndian.PutUint64(payload, uint64(nanos))
		err := v.ctl.ShipTo(v.toAddr, Shipment{Body: &TestBytes{payload}})
		if err != nil {
			panic(err)
		}
		<- v.results
	}
}
