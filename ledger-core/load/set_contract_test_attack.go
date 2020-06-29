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
	a.client = loadgen.NewLoggingHTTPClient(false, 10)
	return nil
}

func (a *SetContractTestAttack) Do(ctx context.Context) loadgen.DoResult {
	var addAmountURL string
	if len(a.GetManager().GeneratorConfig.Generator.Target) == 0 {
		// set default
		addAmountURL = util.GetURL(util.WalletAddAmountPath, "", "")
	} else {
		addAmountURL = a.GetManager().GeneratorConfig.Generator.Target + util.WalletAddAmountPath
	}

	for _, reference := range loadgen.DefaultReadCSV(a) {
		err := util.AddAmountToWallet(a.client, addAmountURL, reference, 100)
		if err != nil {
			return loadgen.DoResult{
				Error:        err,
				RequestLabel: GetContractTestLabel,
			}
		}
	}
	return loadgen.DoResult{
		RequestLabel: SetContractTestLabel,
	}
}
func (a *SetContractTestAttack) Clone(r *loadgen.Runner) loadgen.Attack {
	return &SetContractTestAttack{WithRunner: loadgen.WithRunner{R: r}}
}
