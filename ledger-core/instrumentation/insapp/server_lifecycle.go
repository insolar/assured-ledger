// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insapp

import (
	"context"
	"strconv"
	"sync"

	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type appEntry struct {
	ctx    context.Context
	cm     *component.Manager
	stopFn func()
}

type multiLifecycle struct {
	state atomickit.StartStopFlag
	mutex sync.RWMutex

	initer    *AppInitializer
	apps      map[string]*appEntry
	loggerFn  LoggerInitFunc
	networkFn NetworkInitFunc
	baseCtx   context.Context
}

func (p *multiLifecycle) Start(ctx context.Context, fn errorFunc) {
	p.mutex.RLock() // protect the map
	defer p.mutex.RUnlock()

	if len(p.apps) == 0 {
		panic(throw.IllegalState())
	}

	if !p.state.DoStart(func() {
		for name, app := range p.apps {
			if err := app.cm.Start(ctx); err != nil {
				fn(name, err)
			}
		}
	}) {
		panic(throw.IllegalState())
	}
}

func (p *multiLifecycle) Stop(ctx context.Context, fn errorFunc) {
	if !p.state.DoStop(func() {
		p.mutex.RLock() // protect the map
		defer p.mutex.RUnlock()

		for name, app := range p.apps {
			if err := app.cm.Stop(ctx); err != nil {
				fn(name, err)
			}
		}
	}) {
		panic(throw.IllegalState())
	}
}

func (p *multiLifecycle) StopGraceful(ctx context.Context, fn errorFunc) {
	switch isActive, wasStarted := p.state.Status(); {
	case !wasStarted:
		panic(throw.IllegalState())
	case !isActive:
		// is stopping
		return
	}

	p.mutex.RLock() // protect the map
	defer p.mutex.RUnlock()

	for name, app := range p.apps {
		if err := app.cm.GracefulStop(ctx); err != nil {
			fn(name, err)
		}
	}
}

// func (p *multiLifecycle) addApp()

func (p *multiLifecycle) addInitApp(cfg configuration.Configuration, logger log.Logger) {
	if p.state.WasStarted() {
		panic(throw.IllegalState())
	}
	p._addApp(cfg, logger)
}

func (p *multiLifecycle) _addApp(cfg configuration.Configuration, _ log.Logger) {
	app := &appEntry{}
	app.cm, app.stopFn = p.initer.StartComponents(p.baseCtx, cfg, p.networkFn,
		func(baseCtx context.Context, cfg configuration.Log, nodeRef, nodeRole string) context.Context {
			ctx, _ := inslogger.InitNodeLogger(baseCtx, cfg, nodeRef, nodeRole)
			app.ctx = ctx
			return ctx
		})

	if p.apps == nil {
		p.apps = map[string]*appEntry{}
	}

	name := strconv.Itoa(len(p.apps) + 1)
	p.apps[name] = app
}

/********************************************/

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

