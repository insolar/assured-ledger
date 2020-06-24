package load

import (
	"context"
	"errors"
	"github.com/insolar/assured-ledger/ledger-core/load/util"
	"math/rand"
	"time"

	"github.com/skudasov/loadgen"
)

type CreateContractTestAttack struct {
	loadgen.WithRunner
}

func (a *CreateContractTestAttack) Setup(hc loadgen.RunnerConfig) error {
	a.GetRunner().L.Infof("Setup attacker")
	rand.Seed(time.Now().Unix())
	util.HttpClient = util.CreateHTTPClient()
	return nil
}
func (a *CreateContractTestAttack) Do(ctx context.Context) loadgen.DoResult {
	time.Sleep(400 * time.Millisecond)
	url := util.GetURL(util.WalletCreatePath, "", "")
	rawResp, err := util.SendAPIRequest(url, nil)
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

func (a *CreateContractTestAttack) PutData(mo interface{}) error {
	if mo != nil && a.R.Config.StoreData {
		data := []string{mo.(*util.WalletCreateResponse).Ref}
		loadgen.DefaultWriteCSV(a, data)
	}
	return nil
}
