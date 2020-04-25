// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package handle

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/pkg/errors"
	"go.opencensus.io/stats"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/bus"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/bus/meta"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/flow"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/insmetrics"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger/light/proc"
)

type Init struct {
	dep     *proc.Dependencies
	message *message.Message
	sender  bus.Sender
}

func NewInit(dep *proc.Dependencies, sender bus.Sender, msg *message.Message) *Init {
	return &Init{
		dep:     dep,
		sender:  sender,
		message: msg,
	}
}

func (s *Init) Future(ctx context.Context, f flow.Flow) error {
	return f.Migrate(ctx, s.Present)
}

func (s *Init) Present(ctx context.Context, f flow.Flow) error {
	logger := inslogger.FromContext(ctx)
	err := s.handle(ctx, f)
	if err != nil {
		if err == flow.ErrCancelled {
			logger.Info("Handling flow cancelled")
		} else {
			logger.Error("Handling error: ", err.Error())
		}
	}
	return err
}

func (s *Init) handle(ctx context.Context, _ flow.Flow) error {
	var err error

	meta := payload.Meta{}
	err = meta.Unmarshal(s.message.Payload)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal meta")
	}
	payloadType, err := payload.UnmarshalType(meta.Payload)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal payload type")
	}

	logger := inslogger.FromContext(ctx)
	logger.Debug("Start to handle new message")

	err = fmt.Errorf("no handler for message type %s", payloadType.String())
	bus.ReplyError(ctx, s.sender, meta, err)
	s.errorMetrics(ctx, payloadType.String(), err)
	return err
}

func (s *Init) errorMetrics(ctx context.Context, msgType string, err error) {
	if err == nil {
		return
	}
	errCode := payload.CodeUnknown
	if err == flow.ErrCancelled {
		errCode = payload.CodeFlowCanceled
	}
	cause := errors.Cause(err)
	insError, ok := cause.(*payload.CodedError)
	if ok {
		errCode = insError.GetCode()
	}

	ctx = insmetrics.InsertTag(ctx, KeyErrorCode, errCode.String())
	ctx = insmetrics.InsertTag(ctx, KeyMsgType, msgType)
	stats.Record(ctx, statHandlerError.M(1))
}

func (s *Init) Past(ctx context.Context, f flow.Flow) error {
	msgType := s.message.Metadata.Get(meta.Type)
	if msgType != "" {
		return s.Present(ctx, f)
	}

	meta := payload.Meta{}
	err := meta.Unmarshal(s.message.Payload)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal meta")
	}

	bus.ReplyError(ctx, s.sender, meta, flow.ErrCancelled)
	return nil
}
