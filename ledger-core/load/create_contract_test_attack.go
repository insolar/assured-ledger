package load

import (
	"context"
	"net/http"

	"github.com/skudasov/loadgen"

	"github.com/insolar/assured-ledger/ledger-core/load/util"
)

type CreateContractTestAttack struct {
	loadgen.WithRunner
	client *http.Client
}

func (a *CreateContractTestAttack) Setup(hc loadgen.RunnerConfig) error {
	a.client = loadgen.NewLoggingHTTPClient(a.GetManager().SuiteConfig.DumpTransport, 60)
	return nil
}
func (a *CreateContractTestAttack) Do(ctx context.Context) loadgen.DoResult {
	url := a.GetManager().GeneratorConfig.Generator.Target + util.WalletCreatePath
	walletRef, err := util.CreateSimpleWallet(a.client, url)
	if err != nil {
		return loadgen.DoResult{
			Error:        err,
			RequestLabel: CreateContractTestLabel,
		}
	}
	err = a.PutData(walletRef)
	return loadgen.DoResult{
		RequestLabel: CreateContractTestLabel,
	}
}
func (a *CreateContractTestAttack) Clone(r *loadgen.Runner) loadgen.Attack {
	return &CreateContractTestAttack{WithRunner: loadgen.WithRunner{R: r}}
}

func (a *CreateContractTestAttack) PutData(reference string) error {
	if a.R.Config.StoreData {
		loadgen.DefaultWriteCSV(a, []string{reference})
	}
	return nil
}
