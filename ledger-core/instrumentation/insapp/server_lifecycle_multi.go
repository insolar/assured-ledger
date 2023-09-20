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
	multiFn   MultiNodeConfigFunc
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

func (p *multiLifecycle) addInitApp(cfg configuration.Configuration, _ log.Logger) {
	if p.state.WasStarted() {
		panic(throw.IllegalState())
	}
	name := strconv.Itoa(len(p.apps) + 1)
	p._addApp(name, cfg)
}

func (p *multiLifecycle) _addApp(name string, cfg configuration.Configuration) *appEntry {
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

	p.apps[name] = app
	return app
}

func (p *multiLifecycle) AppStop(s string) (bool, error) {
	return p.appStop(s, false)
}

func (p *multiLifecycle) AppStopGraceful(s string) (bool, error) {
	return p.appStop(s, true)
}

func (p *multiLifecycle) appStop(s string, graceful bool) (bool, error) {
	if !p.state.IsActive() {
		panic(throw.IllegalState())
	}

	app := p.takeOut(s)
	if app == nil {
		return false, nil
	}

	if graceful {
		if err := app.cm.GracefulStop(app.ctx); err != nil {
			return true, err
		}
	}

	if app.stopFn != nil {
		app.stopFn()
	}
	err := app.cm.Stop(app.ctx)
	return true, err
}

func (p *multiLifecycle) AppStart(s string) (bool, error) {
	if s == "" {
		panic(throw.IllegalValue())
	}

	switch app, start := p.addLateApp(s); {
	case app == nil:
		return false, nil
	case start:
		err := app.cm.Start(app.ctx)
		return true, err
	default:
		return true, nil
	}
}

func (p *multiLifecycle) addLateApp(s string) (*appEntry, bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	isActive, wasStarted := p.state.Status()
	if wasStarted && !isActive {
		// stopped
		panic(throw.IllegalState())
	}

	if _, ok := p.apps[s]; ok {
		return nil, false
	}

	cfg := p.initer.confProvider.GetNamedConfig(s)

	return p._addApp(s, cfg), isActive
}

func (p *multiLifecycle) takeOut(name string) *appEntry {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	app := p.apps[name]
	delete(p.apps, name)
	return app
}

