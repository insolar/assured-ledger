// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package appfoundation

import "github.com/insolar/assured-ledger/ledger-core/v2/reference"

type SagaAcceptInfo struct {
	Amount     string
	FromMember reference.Global
	Request    reference.Global
}
