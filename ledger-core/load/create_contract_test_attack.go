package load

import (
	"context"
	"errors"
	"github.com/insolar/assured-ledger/ledger-core/load/util"
	"github.com/skudasov/loadgen"
	"time"
)

type CreateContractTestAttack struct {
	loadgen.WithRunner
}

func (a *CreateContractTestAttack) Setup(hc loadgen.RunnerConfig) error {
	util.HttpClient = util.CreateHTTPClient()
	return nil
}
func (a *CreateContractTestAttack) Do(ctx context.Context) loadgen.DoResult {
	time.Sleep(400 * time.Millisecond)
	var getBalanceURL string
	if len(a.GetManager().GeneratorConfig.Generator.Target) == 0 {
		// set default
		getBalanceURL = util.GetURL(util.WalletCreatePath, "", "")
	} else {
		getBalanceURL = a.GetManager().GeneratorConfig.Generator.Target + util.WalletCreatePath
	}
	rawResp, err := util.SendAPIRequest(getBalanceURL, nil)
	if err != nil {
		a.GetRunner().L.Error(err)
		return loadgen.DoResult{
			Error:        err,
			RequestLabel: CreateContractTestLabel,
		}
	}
	resp, err := util.UnmarshalWalletCreateResponse(rawResp)
	if err != nil {
		a.GetRunner().L.Error(err)
		return loadgen.DoResult{
			Error:        err,
			RequestLabel: CreateContractTestLabel,
		}
	}
	// store result
	_ = a.PutData(resp)
	return loadgen.DoResult{
		Error:        errors.New(""),
		RequestLabel: CreateContractTestLabel,
	}
}
func (a *CreateContractTestAttack) Clone(r *loadgen.Runner) loadgen.Attack {
	return &CreateContractTestAttack{WithRunner: loadgen.WithRunner{R: r}}
}

func (a *CreateContractTestAttack) PutData(mo util.WalletCreateResponse) error {
	if a.R.Config.StoreData {
		data := []string{mo.Ref}
		loadgen.DefaultWriteCSV(a, data)
	}
	return nil
}
