// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/insolar/blob/master/LICENSE.md.

package facade

import (
	"context"
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/bus"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/pkg/errors"
)

type Marshaler interface {
	Marshal() ([]byte, error)
	// Temporary commented to look like insolar.Payload
	//MarshalHead() ([]byte, error)
	//MarshalContent() ([]byte, error)
}

type option struct {
	syncBody bool
}

type SendOption func(*option)

func WithSyncBody() SendOption {
	return func(o *option) {
		o.syncBody = true
	}
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/network/facade.Messenger -o ./ -s _mock.go -g

type Messenger interface {
	// blocks if network unreachable
	SendRole(ctx context.Context, msg Marshaler, role insolar.DynamicRole, object insolar.Reference, pn pulse.Number, opts ...SendOption) error
	SendTarget(ctx context.Context, msg Marshaler, target insolar.Reference, opts ...SendOption) error
}

func apply(opts ...SendOption) {
	defaultOpts := &option{}
	for _, opt := range opts {
		opt(defaultOpts)
	}

	fmt.Printf("%#v", defaultOpts)

}

type DefaultMessenger struct {
	sender bus.Sender
}

func NewDefaultMessenger(sender bus.Sender) *DefaultMessenger {
	return &DefaultMessenger{sender: sender}
}

func main() {
	// bus := bus.NewBus(configuration.NewBus(), nil, nil, nil, nil)
	// serv := CreateBusService(NewDefaultMessenger(bus))
	//
	// pl := payload.Replication{}
	// serv.PrepareAsync(nil, func(svc MessengerService) smachine.AsyncResultFunc {
	// 	err := svc.SendRole(context.Background(), &pl, insolar.DynamicRoleHeavyExecutor, gen.Reference(), gen.PulseNumber())
	// 	if err != nil {
	// 		panic(err)
	// 	}
	//
	// 	return func(ctx smachine.AsyncResultContext) {
	// 		ctx.GetContext()
	// 	}
	// })
}

func joinOptions(opts ...SendOption) *option {
	emptyOpts := &option{}
	for _, opt := range opts {
		opt(emptyOpts)
	}

	return emptyOpts
}

func (dm *DefaultMessenger) SendRole(ctx context.Context, msg Marshaler, role insolar.DynamicRole, object insolar.Reference, pn pulse.Number, opts ...SendOption) error {
	_ = joinOptions(opts...)

	waterMillMsg, err := payload.NewMessage(msg.(insolar.Payload))
	if err != nil {
		return errors.Wrap(err, "Can't create watermill message")
	}

	_, done := dm.sender.SendRole(ctx, waterMillMsg, role, object)
	done()
	return nil
}

func (dm *DefaultMessenger) SendTarget(ctx context.Context, msg Marshaler, target insolar.Reference, opts ...SendOption) error {
	_ = joinOptions(opts...)

	waterMillMsg, err := payload.NewMessage(msg.(insolar.Payload))
	if err != nil {
		return errors.Wrap(err, "Can't create watermill message")
	}

	_, done := dm.sender.SendTarget(ctx, waterMillMsg, target)
	done()
	return nil
}
