// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package handle

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/flow"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
)

type Error struct {
	message *message.Message
}

func NewError(msg *message.Message) *Error {
	return &Error{
		message: msg,
	}
}

func (s *Error) Present(ctx context.Context, f flow.Flow) error {
	pl, err := payload.UnmarshalFromMeta(s.message.Payload)
	if err != nil {
		inslogger.FromContext(ctx).Error(errors.Wrap(err, "failed to unmarshal error"))
		return nil
	}
	p, ok := pl.(*payload.Error)
	if !ok {
		inslogger.FromContext(ctx).Errorf("unexpected error type %T", pl)
		return nil
	}

	inslogger.FromContext(ctx).Error("received error: ", p.Text)
	return nil
}
