// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package termination

import (
	"context"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network/storage"
	"github.com/insolar/assured-ledger/ledger-core/pulse"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
)

type Handler struct {
	sync.Mutex
	done        chan struct{}
	terminating bool

	Leaver        node.Leaver
	PulseAccessor storage.PulseAccessor `inject:""`
}

func NewHandler(l node.Leaver) *Handler {
	return &Handler{Leaver: l}
}

// TODO take ETA by role of node
func (t *Handler) Leave(ctx context.Context, leaveAfterPulses pulse.Number) {
	doneChan := t.leave(ctx, leaveAfterPulses)
	<-doneChan
}

func (t *Handler) leave(ctx context.Context, leaveAfterPulses pulse.Number) chan struct{} {
	t.Lock()
	defer t.Unlock()

	if !t.terminating {
		t.terminating = true
		t.done = make(chan struct{}, 1)

		if leaveAfterPulses == 0 {
			inslogger.FromContext(ctx).Debug("Handler.Leave() with 0")
			t.Leaver.Leave(ctx, 0)
		} else {
			pulse, err := t.PulseAccessor.GetLatestPulse(ctx)
			if err != nil {
				inslogger.FromContext(ctx).Panicf("smth goes wrong. There is no pulse in the storage. err - %v", err)
			}
			pulseDelta := pulse.NextPulseNumber - pulse.PulseNumber

			inslogger.FromContext(ctx).Debugf("Handler.Leave() with leaveAfterPulses: %+v, in pulse %+v", leaveAfterPulses, pulse.PulseNumber+leaveAfterPulses*pulseDelta)
			t.Leaver.Leave(ctx, pulse.PulseNumber+leaveAfterPulses*pulseDelta)
		}
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
