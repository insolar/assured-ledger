// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build slowtest

package grand

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
)

// BenchmarkUserCreation benchmarks parallel user creation (with maximum 100 workers at a time)
func BenchmarkUserCreation(b *testing.B) {
	ctx := context.Background()
	cfg := DefaultVMConfig()

	s, err := NewVirtualServer(b, ctx, cfg).WithGenesis().PrepareAndStart()
	require.NoError(b, err)
	defer s.Stop(ctx)

	var (
		iterations = b.N
		helper     = ServerHelper{s}
		syncAssert = NewTSAssert(b)
		sema       = semaphore.NewWeighted(100)
	)

	b.ResetTimer()

	var wg sync.WaitGroup
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func() {
			defer wg.Done()

			err := sema.Acquire(ctx, 1)
			if err == nil {
				defer sema.Release(1)
			} else {
				panic(fmt.Sprintf("unexpected: %s", err.Error()))
			}

			_, err = helper.createUser(ctx)
			syncAssert.NoError(err, "failed to create user")
		}()
	}
	wg.Wait()
}
