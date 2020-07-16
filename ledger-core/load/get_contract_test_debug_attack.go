package load

import (
	"context"
	"net/http"

	"github.com/insolar/loadgenerator"

	"github.com/insolar/assured-ledger/ledger-core/load/util"
)

type GetContractTestDebugAttack struct {
	loadgen.WithRunner
	client *http.Client
}

func (a *GetContractTestDebugAttack) Setup(hc loadgen.RunnerConfig) error {
	a.client = loadgen.NewLoggingHTTPClient(a.GetManager().SuiteConfig.DumpTransport, 60)
	return nil
}
func (a *GetContractTestDebugAttack) Do(ctx context.Context) loadgen.DoResult {
	url := a.GetManager().GeneratorConfig.Generator.Target
	err := util.GetDebug200(a.client, url)
	if err != nil {
		return loadgen.DoResult{
			Error:        err,
			RequestLabel: GetContractTestDebugLabel,
		}
	}
	return loadgen.DoResult{
		RequestLabel: GetContractTestDebugLabel,
	}
}
func (a *GetContractTestDebugAttack) Clone(r *loadgen.Runner) loadgen.Attack {
	return &GetContractTestDebugAttack{WithRunner: loadgen.WithRunner{R: r}}
}
