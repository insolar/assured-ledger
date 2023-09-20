package insapp

import (
	"context"

	"github.com/insolar/component-manager"
)

type monoLifecycle struct {
	cm     *component.Manager
	stopFn func()
}

func (m monoLifecycle) Start(ctx context.Context, fn errorFunc) {
	if err := m.cm.Start(ctx); err != nil {
		fn("", err)
	}
}

func (m monoLifecycle) Stop(ctx context.Context, fn errorFunc) {
	if m.stopFn != nil {
		m.stopFn()
	}

	if err := m.cm.Stop(ctx); err != nil {
		fn("", err)
	}
}

func (m monoLifecycle) StopGraceful(ctx context.Context, fn errorFunc) {
	if err := m.cm.GracefulStop(ctx); err != nil {
		fn("", err)
	}
}

