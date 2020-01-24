//
//    Copyright 2019 Insolar Technologies
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.
//

package smachine

import (
	"context"
	"math"
)

func StartChannelWorker(ctx context.Context, ch <-chan AdapterCall, runArg interface{}) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case t, ok := <-ch:
				if !ok {
					return
				}
				if err := t.RunAndSendResult(runArg); err != nil {
					t.ReportError(err)
				}
			}
		}
	}()
}

func StartChannelWorkerParallelCalls(ctx context.Context, parallelLimit uint16, ch <-chan AdapterCall, runArg interface{}) {
	switch parallelLimit {
	case 1:
		StartChannelWorker(ctx, ch, runArg)
		return
	case 0, math.MaxInt16:
		startChannelWorkerUnlimParallel(ctx, ch, runArg)
	default:
		for i := parallelLimit; i > 0; i-- {
			StartChannelWorker(ctx, ch, runArg)
		}
	}
}

func startChannelWorkerUnlimParallel(ctx context.Context, ch <-chan AdapterCall, runArg interface{}) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case t, ok := <-ch:
				if !ok {
					return
				}
				go func() {
					if err := t.RunAndSendResult(runArg); err != nil {
						t.ReportError(err)
					}
				}()
			}
		}
	}()
}
