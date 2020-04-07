// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package ratelimiter

import (
	"context"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

func NewController(trafficUnit uint32, refillPeriod time.Duration) Controller {
	switch {
	case trafficUnit <= 0:
		panic(throw.IllegalValue())
	case refillPeriod <= 0:
		panic(throw.IllegalValue())
	}
	c := Controller{}
	c.root.amountScale = trafficUnit
	c.period = refillPeriod
	c.root.init()
	return c
}

type Controller struct {
	root     PeriodManager
	period   time.Duration
	throttle atomickit.Uint32
	started  bool
}

func (p *Controller) Init() {
	if p.started {
		panic(throw.IllegalState())
	}
	p.root.init()
}

func (p *Controller) SetIncrement(level uint32) {
	p.throttle.Store(level)
}

func (p *Controller) GetIncrement() uint32 {
	return p.throttle.Load()
}

func (p *Controller) Start(ctx context.Context) {
	if p.started {
		panic(throw.IllegalState())
	}
	if ctx == nil {
		panic(throw.IllegalValue())
	}
	p.started = true
	go p.run(ctx)
}

func (p *Controller) Root() *PeriodManager {
	return &p.root
}

func (p *Controller) run(ctx context.Context) {
	ticker := time.Tick(p.period)
	for {
		select {
		case <-ticker:
			step := p.throttle.Load()
			if step > 0 {
				p.root.nextPeriod(uint(step))
			}
		case <-ctx.Done():
			return
		}
	}
}
