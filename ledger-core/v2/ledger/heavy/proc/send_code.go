// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package proc

import (
	"context"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/bus"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger/object"
)

type SendCode struct {
	message payload.Meta

	Dep struct {
		RecordAccessor object.RecordAccessor
		Sender         bus.Sender
	}
}

func NewSendCode(msg payload.Meta) *SendCode {
	return &SendCode{
		message: msg,
	}
}

func (p *SendCode) Proceed(ctx context.Context) error {
	getCode := payload.GetCode{}
	err := getCode.Unmarshal(p.message.Payload)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal GetCode message")
	}

	rec, err := p.Dep.RecordAccessor.ForID(ctx, getCode.CodeID)
	if err != nil {
		return errors.Wrap(err, "failed to fetch record")
	}
	buf, err := rec.Marshal()
	if err != nil {
		return errors.Wrap(err, "failed to marshal record")
	}
	msg, err := payload.NewMessage(&payload.Code{
		Record: buf,
	})
	if err != nil {
		return errors.Wrap(err, "failed to create message")
	}

	p.Dep.Sender.Reply(ctx, p.message, msg)

	return nil
}
