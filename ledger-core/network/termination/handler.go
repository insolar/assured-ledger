// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package termination

import (
	"context"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

type Handler struct {
	sync.Mutex
	done        chan struct{}
	terminating bool

	Leaver        Leaver
	PulseAccessor beat.History `inject:""`
}

func NewHandler(l Leaver) *Handler {
	return &Handler{Leaver: l}
}

// TODO take ETA by role of node
func (t *Handler) Leave(ctx context.Context, leaveAfter pulse.Number) {
	doneChan := t.leave(ctx, leaveAfter)
	<-doneChan
}

func (t *Handler) leave(ctx context.Context, leaveAfter pulse.Number) chan struct{} {
	t.Lock()
	defer t.Unlock()

	if !t.terminating {
		t.terminating = true
		t.done = make(chan struct{}, 1)
		t.Leaver.Leave(ctx, leaveAfter)
	}

	return t.done
}

func (t *Handler) OnLeaveApproved(ctx context.Context) {
	t.Lock()
	defer t.Unlock()
	if t.terminating {
		inslogger.FromContext(ctx).Debug("Handler.OnLeaveApproved() received")
		t.terminating = false
		close(t.done)
	}
}

func (t *Handler) Abort(ctx context.Context, reason string) {
	inslogger.FromContext(ctx).Fatal(reason)
}

func (t *Handler) Terminating() bool {
	return t.terminating
}
