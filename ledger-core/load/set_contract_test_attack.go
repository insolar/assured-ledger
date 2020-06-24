package load

import (
	"context"
	"errors"
	"github.com/insolar/assured-ledger/ledger-core/load/util"
	"github.com/skudasov/loadgen"
)

type SetContractTestAttack struct {
	loadgen.WithRunner
}

func (a *SetContractTestAttack) Setup(hc loadgen.RunnerConfig) error {
	util.HttpClient = util.CreateHTTPClient()
	return nil
}
func (a *SetContractTestAttack) Do(ctx context.Context) loadgen.DoResult {
	if len(loadgen.DefaultReadCSV(a)) == 0 {
		return loadgen.DoResult{
			Error:        errors.New("not data for test"),
			RequestLabel: GetContractTestLabel,
		}
	}
	var addAmountURL string
	if len(a.GetManager().GeneratorConfig.Generator.Target) == 0 {
		// set default
		addAmountURL = util.GetURL(util.WalletAddAmountPath, "", "")
	} else {
		addAmountURL = a.GetManager().GeneratorConfig.Generator.Target + util.WalletAddAmountPath
	}

	for _, reference := range loadgen.DefaultReadCSV(a) {
		err := util.AddAmountToWallet(addAmountURL, reference, 100)
		if err != nil {
			return loadgen.DoResult{
				Error:        err,
				RequestLabel: GetContractTestLabel,
			}
		}
	}
	return loadgen.DoResult{
		Error:        nil,
		RequestLabel: SetContractTestLabel,
	}
}
func (a *SetContractTestAttack) Clone(r *loadgen.Runner) loadgen.Attack {
	return &SetContractTestAttack{WithRunner: loadgen.WithRunner{R: r}}
}
