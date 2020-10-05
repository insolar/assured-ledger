// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"fmt"
	"math"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l2/uniserver"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
)

func TestMassiveSend(t *testing.T) {
	// TODO https://insolar.atlassian.net/browse/PLAT-826
	// workaround with set max value for case above
	maxReceiveExceptions = math.MaxInt64
	defer func() {
		maxReceiveExceptions = 1 << 8
	}()

	cfg := uniserver.ServerConfig{
		BindingAddress: "127.0.0.1:0",
		UDPMaxSize:     1400,
		UDPParallelism: 2,
		PeerLimit:      -1,
	}

	numberShipments := 10000

	var receive1 atomickit.Uint32
	var receive2 atomickit.Uint32
	results := make(chan string, 2000)

	h := NewUnitProtoServersHolder(TestLogAdapter{t: t})
	defer h.stop()

	_, err := h.createService(cfg, func(a ReturnAddress, _ nwapi.PayloadCompleteness, v interface{}) error {
		receive1.Add(1)
		// t.Log(a.String(), "Ctl1:", v)
		// fmt.Println(a.String(), "Ctl1:", v)
		s := v.(fmt.Stringer).String()
		results <- s
		// s := v.(fmt.Stringer).String() + "-return"
		// return ctl1.ShipReturn(a, Shipment{Head: &TestString{s}})
		return nil
	})
	require.NoError(t, err)

	srv2, err := h.createService(cfg, func(a ReturnAddress, _ nwapi.PayloadCompleteness, v interface{}) error {
		receive2.Add(1)
		s := v.(fmt.Stringer).String()
		// fmt.Println(a.String(), "Ctl2:", s)
		// t.Log(a.String(), "Ctl2:", s)
		results <- s
		return nil
	})
	require.NoError(t, err)

	// loopback
	err = srv2.service.ShipTo(NewDirectAddress(2), Shipment{Head: &TestString{"loc0"}})
	require.NoError(t, err)

	err = srv2.service.ShipTo(NewDirectAddress(1), Shipment{Head: &TestString{"rem0"}})
	require.NoError(t, err)

	require.Equal(t, "loc0", <-results)
	require.Equal(t, "rem0", <-results)
	// require.Equal(t, "rem0-return", <-results)

	sentMap := sync.Map{}
	totalCount := atomickit.Uint32{}

	for i := 1; i <= numberShipments; i++ {
		// loopback
		// err = ctl2.ShipTo(NewDirectAddress(2), Shipment{Head: &TestString{"loc" + strconv.Itoa(i)}})
		// require.NoError(t, err)

		key := "rem" + strconv.Itoa(i)
		sentMap.Store(key, nil)
		totalCount.Add(1)
		err = srv2.service.ShipTo(NewDirectAddress(1), Shipment{Head: &TestString{key}})
		require.NoError(t, err)
	}

	go func() {
		for totalCount.Load() > 0 {
			key := <-results
			sentMap.Delete(key)
			totalCount.Sub(1)
		}
	}()

	time.Sleep(time.Second * 10)

	if received := totalCount.Load(); received > 0 {
		t.Logf("Not all message received, sended:%d reseived:%d", numberShipments, received)
		for i := 0; i < 10; {
			t.Logf("Await iteration %d", i)
			if totalCount.Load() > 0 {
				time.Sleep(time.Second * 10)
				i++
			} else {
				break
			}
		}
	}

	if n := totalCount.Load(); n > 0 {
		sentMap.Range(func(key, value interface{}) bool {
			fmt.Println(key)
			return true
		})

		require.Zero(t, n)
	}
}
