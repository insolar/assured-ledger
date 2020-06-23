package load

import (
	"context"
	"time"

	"github.com/skudasov/loadgen"
)

type SetContractTestAttack struct {
	loadgen.WithRunner
}

func (a *SetContractTestAttack) Setup(hc loadgen.RunnerConfig) error {
	return nil
}
func (a *SetContractTestAttack) Do(ctx context.Context) loadgen.DoResult {
	time.Sleep(200 * time.Millisecond)
	return loadgen.DoResult{
		Error:        nil,
		RequestLabel: SetContractTestLabel,
	}
}
func (a *SetContractTestAttack) Clone(r *loadgen.Runner) loadgen.Attack {
	return &SetContractTestAttack{WithRunner: loadgen.WithRunner{R: r}}
}
