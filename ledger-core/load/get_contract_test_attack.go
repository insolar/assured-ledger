package load

import (
	"context"
	"time"

	"github.com/skudasov/loadgen"
)

type GetContractTestAttack struct {
	loadgen.WithRunner
}

func (a *GetContractTestAttack) Setup(hc loadgen.RunnerConfig) error {
	return nil
}
func (a *GetContractTestAttack) Do(ctx context.Context) loadgen.DoResult {
	time.Sleep(1 * time.Second)
	return loadgen.DoResult{
		Error:        nil,
		RequestLabel: GetContractTestLabel,
	}
}
func (a *GetContractTestAttack) Clone(r *loadgen.Runner) loadgen.Attack {
	return &GetContractTestAttack{WithRunner: loadgen.WithRunner{R: r}}
}
