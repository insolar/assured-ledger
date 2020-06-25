package load

import (
	"context"
	"github.com/insolar/assured-ledger/ledger-core/load/util"
	"github.com/skudasov/loadgen"
)

type CreateContractTestAttack struct {
	loadgen.WithRunner
}

func (a *CreateContractTestAttack) Setup(hc loadgen.RunnerConfig) error {
	util.HttpClient = util.CreateHTTPClient()
	return nil
}
func (a *CreateContractTestAttack) Do(ctx context.Context) loadgen.DoResult {
	var walletCreateURL string
	if len(a.GetManager().GeneratorConfig.Generator.Target) == 0 {
		// set default
		walletCreateURL = util.GetURL(util.WalletCreatePath, "", "")
	} else {
		walletCreateURL = a.GetManager().GeneratorConfig.Generator.Target + util.WalletCreatePath
	}
	walletRef, err := util.CreateSimpleWallet(walletCreateURL)
	if err != nil {
		return loadgen.DoResult{
			Error:        err,
			RequestLabel: GetContractTestLabel,
		}
	}
	// store result
	err = a.PutData(walletRef)
	return loadgen.DoResult{
		Error:        err,
		RequestLabel: CreateContractTestLabel,
	}
}
func (a *CreateContractTestAttack) Clone(r *loadgen.Runner) loadgen.Attack {
	return &CreateContractTestAttack{WithRunner: loadgen.WithRunner{R: r}}
}

func (a *CreateContractTestAttack) PutData(reference string) error {
	if a.R.Config.StoreData {
		data := []string{reference}
		loadgen.DefaultWriteCSV(a, data)
	}
	return nil
}
