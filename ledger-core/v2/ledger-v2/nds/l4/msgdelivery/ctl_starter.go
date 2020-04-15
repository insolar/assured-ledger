// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/uniproto"
)

type protoStarter struct {
	ctl *Controller
}

func (p protoStarter) Start(sender uniproto.Sender) {
	panic("implement me")
}

func (p protoStarter) Stop() {
	panic("implement me")
}
