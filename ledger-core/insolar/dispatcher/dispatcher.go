// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package dispatcher

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulsestor"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/insolar/dispatcher.Dispatcher -o ./ -s _mock.go -g
type Dispatcher interface {
	BeginPulse(ctx context.Context, pulse pulsestor.Pulse)
	ClosePulse(ctx context.Context, pulse pulsestor.Pulse)
	Process(msg *message.Message) error
}
