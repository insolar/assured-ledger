package load

import (
	"context"
	"github.com/insolar/assured-ledger/ledger-core/load/util"
	"github.com/pkg/errors"
	"github.com/skudasov/loadgen"
)

type GetContractTestAttack struct {
	loadgen.WithRunner
}

func (a *GetContractTestAttack) Setup(hc loadgen.RunnerConfig) error {
	util.HttpClient = util.CreateHTTPClient()
	return nil
}
func (a *GetContractTestAttack) Do(ctx context.Context) loadgen.DoResult {
	if len(loadgen.DefaultReadCSV(a)) == 0 {
		return loadgen.DoResult{
			Error:        errors.New("not data for test"),
			RequestLabel: GetContractTestLabel,
		}
	}
	var getBalanceURL string
	if len(a.GetManager().GeneratorConfig.Generator.Target) == 0 {
		// set default
		getBalanceURL = util.GetURL(util.WalletGetBalancePath, "", "")
	} else {
		getBalanceURL = a.GetManager().GeneratorConfig.Generator.Target + util.WalletGetBalancePath
	}
	// TODO: should we iterate over all csv file or work with only first reference from file?
	for _, reference := range loadgen.DefaultReadCSV(a) {
		balance, err := util.GetWalletBalance(getBalanceURL, reference)
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
	}
	return loadgen.DoResult{
		Error:        nil,
		RequestLabel: GetContractTestLabel,
	}
}
func (a *GetContractTestAttack) Clone(r *loadgen.Runner) loadgen.Attack {
	return &GetContractTestAttack{WithRunner: loadgen.WithRunner{R: r}}
}
