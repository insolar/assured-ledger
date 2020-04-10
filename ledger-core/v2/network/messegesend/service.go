// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/insolar/blob/master/LICENSE.md.

package messegesend

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/bus"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"

	"github.com/pkg/errors"
)

type options struct {
	syncBody bool
}

func (o *options) applyOptions(opts ...SendOption) {
	for _, opt := range opts {
		opt(o)
	}
}

type SendOption func(*options)

func WithSyncBody() SendOption {
	return func(o *options) {
		o.syncBody = true
	}
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/network/messegesend.Messenger -o ./ -s _mock.go -g

type Messenger interface {
	// blocks if network unreachable
	SendRole(ctx context.Context, msg payload.Marshaler, role insolar.DynamicRole, object insolar.Reference, pn pulse.Number, opts ...SendOption) error
	SendTarget(ctx context.Context, msg payload.Marshaler, target insolar.Reference, opts ...SendOption) error
}

type Service struct {
	sender bus.Sender
}

func NewService(sender bus.Sender) *Service {
	return &Service{sender: sender}
}

func (dm *Service) SendRole(ctx context.Context, msg payload.Marshaler, role insolar.DynamicRole, object insolar.Reference, pn pulse.Number, opts ...SendOption) error {
	waterMillMsg, err := payload.NewMessage(msg.(payload.Payload))
	if err != nil {
		return errors.Wrap(err, "Can't create watermill message")
	}

	_, done := dm.sender.SendRole(ctx, waterMillMsg, role, object)
	done()
	return nil
}

func (dm *Service) SendTarget(ctx context.Context, msg payload.Marshaler, target insolar.Reference, opts ...SendOption) error {
	waterMillMsg, err := payload.NewMessage(msg.(payload.Payload))
	if err != nil {
		return errors.Wrap(err, "Can't create watermill message")
	}

	_, done := dm.sender.SendTarget(ctx, waterMillMsg, target)
	done()
	return nil
}
