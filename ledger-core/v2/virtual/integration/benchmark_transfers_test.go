//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// +build slowtest

package integration

import (
	"context"
	_ "net/http/pprof"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/testutils"
)

const parallelClientCount = 5

// BenchmarkTransferSimple makes parallelClientCount * 2 wallets and then makes X transfers from one wallet to another
func BenchmarkTransferSimple(b *testing.B) {
	ctx := context.Background()
	cfg := DefaultVMConfig()

	s, err := NewVirtualServer(b, ctx, cfg).WithGenesis().PrepareAndStart()
	require.NoError(b, err)
	defer s.Stop(ctx)

	var (
		iterations = b.N
		helper     = ServerHelper{s}
		syncAssert = assert.New(&testutils.SyncT{TB: b})
	)

	feeWalletBalance, err := helper.getBalance(ctx, *FeeWalletUser)
	require.NoError(b, err)

	var userPairs = make([]struct {
		from, to            *User
		expectedBalanceFrom int64
		expectedBalanceTo   int64
	}, parallelClientCount)

	for i := 0; i < parallelClientCount; i++ {
		userPairs[i].from, err = helper.createUser(ctx)
		require.NoError(b, err)
		userPairs[i].expectedBalanceFrom, err = helper.getBalance(ctx, *userPairs[i].from)
		require.NoError(b, err)

		userPairs[i].to, err = helper.createUser(ctx)
		require.NoError(b, err)
		userPairs[i].expectedBalanceTo, err = helper.getBalance(ctx, *userPairs[i].to)
		require.NoError(b, err)
	}

	{ // run benchmark
		b.ResetTimer()

		var wg sync.WaitGroup
		wg.Add(parallelClientCount)

		for i := 0; i < parallelClientCount; i++ {
			go func(usersPosition int) {
				defer wg.Done()

				pair := &userPairs[usersPosition]

				for iteration := 0; iteration < iterations; iteration++ {
					transferAmount := int64(1000)

					fee, err := helper.transferMoney(ctx, *pair.from, *pair.to, transferAmount)
					if !syncAssert.NoErrorf(err, "failed to transfer money") {
						return
					}

					pair.expectedBalanceFrom -= fee + transferAmount
					pair.expectedBalanceTo += transferAmount
					atomic.AddInt64(&feeWalletBalance, fee)
				}
			}(i)
		}

		wg.Wait()
		b.StopTimer()
	}

	{ // all requests were sent, now we should wait until all saga transfers would be done
		var wg sync.WaitGroup
		wg.Add(parallelClientCount)

		for i := 0; i < parallelClientCount; i++ {
			go func(usersPosition int) {
				defer wg.Done()

				pair := &userPairs[usersPosition]

				balance, err := helper.getBalance(ctx, *pair.from)
				if syncAssert.NoError(err) {
					syncAssert.Equal(balance, pair.expectedBalanceFrom)
				} else {
					return
				}

				err = helper.waitBalance(ctx, *pair.to, pair.expectedBalanceTo)
				syncAssert.NoError(err)
			}(i)
		}

		wg.Wait()

		err = helper.waitBalance(ctx, *FeeWalletUser, feeWalletBalance)
		syncAssert.NoError(err)
	}
}
