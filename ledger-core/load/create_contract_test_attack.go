package load

import (
	"context"
	"errors"
	"time"

	"github.com/skudasov/loadgen"
)

type CreateContractTestAttack struct {
	loadgen.WithRunner
}

func (a *CreateContractTestAttack) Setup(hc loadgen.RunnerConfig) error {
	return nil
}
func (a *CreateContractTestAttack) Do(ctx context.Context) loadgen.DoResult {
	time.Sleep(400 * time.Millisecond)
	return loadgen.DoResult{
		Error:        errors.New(""),
		RequestLabel: CreateContractTestLabel,
	}
}
func (a *CreateContractTestAttack) Clone(r *loadgen.Runner) loadgen.Attack {
	return &CreateContractTestAttack{WithRunner: loadgen.WithRunner{R: r}}
}
