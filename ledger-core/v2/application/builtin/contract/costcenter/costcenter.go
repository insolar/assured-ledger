// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package costcenter

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/builtin/foundation"
)

type CostCenter struct {
	foundation.BaseContract
	FeeMember *insolar.Reference
}

// New creates new CostCenter.
func New(feeMember *insolar.Reference) (*CostCenter, error) {
	return &CostCenter{
		FeeMember: feeMember,
	}, nil
}

// GetFeeMember gets fee member reference.
// ins:immutable
func (cc *CostCenter) GetFeeMember() (*insolar.Reference, error) {
	return cc.FeeMember, nil
}

const Fee = "1000000000"

// CalcFee calculates fee for amount. Returns fee.
// ins:immutable
func (cc *CostCenter) CalcFee(amountStr string) (string, error) {
	return Fee, nil
}
