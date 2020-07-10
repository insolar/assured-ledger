package load

import (
	"context"
	"net/http"

	"github.com/skudasov/loadgen"

	"github.com/insolar/assured-ledger/ledger-core/load/util"
)

type SetContractTestAttack struct {
	loadgen.WithRunner
	client *http.Client
}

func (a *SetContractTestAttack) Setup(hc loadgen.RunnerConfig) error {
	a.client = loadgen.NewLoggingHTTPClient(a.GetManager().SuiteConfig.DumpTransport, 60)
	return nil
}

func (a *SetContractTestAttack) Do(ctx context.Context) loadgen.DoResult {
	url := a.GetManager().GeneratorConfig.Generator.Target + util.WalletAddAmountPath
	reference := loadgen.DefaultReadCSV(a)
	err := util.AddAmountToWallet(a.client, url, reference[0], 100)
	if err != nil {
		return loadgen.DoResult{
			Error:        err,
			RequestLabel: SetContractTestLabel,
		}
	}

	return loadgen.DoResult{
		RequestLabel: SetContractTestLabel,
	}
}
func (a *SetContractTestAttack) Clone(r *loadgen.Runner) loadgen.Attack {
	return &SetContractTestAttack{WithRunner: loadgen.WithRunner{R: r}}
}
