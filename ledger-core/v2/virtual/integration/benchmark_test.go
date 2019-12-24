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
