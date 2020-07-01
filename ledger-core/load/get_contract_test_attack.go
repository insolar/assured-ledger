package load

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"github.com/skudasov/loadgen"

	"github.com/insolar/assured-ledger/ledger-core/load/util"
)

type GetContractTestAttack struct {
	loadgen.WithRunner
	client *http.Client
}

func (a *GetContractTestAttack) Setup(hc loadgen.RunnerConfig) error {
	a.client = loadgen.NewLoggingHTTPClient(a.GetManager().SuiteConfig.DumpTransport, 60)
	return nil
}
func (a *GetContractTestAttack) Do(ctx context.Context) loadgen.DoResult {
	url := a.GetManager().GeneratorConfig.Generator.Target + util.WalletGetBalancePath
	reference := loadgen.DefaultReadCSV(a)
	balance, err := util.GetWalletBalance(a.client, url, reference[0])
	if err != nil {
		return loadgen.DoResult{
			Error:        err,
			RequestLabel: GetContractTestLabel,
		}
	}
	if balance != util.StartBalance {
		return loadgen.DoResult{
			Error:        errors.New("balance is not equal to start balance"),
			RequestLabel: GetContractTestLabel,
		}
	}

	return loadgen.DoResult{
		RequestLabel: GetContractTestLabel,
	}
}
func (a *GetContractTestAttack) Clone(r *loadgen.Runner) loadgen.Attack {
	return &GetContractTestAttack{WithRunner: loadgen.WithRunner{R: r}}
}
