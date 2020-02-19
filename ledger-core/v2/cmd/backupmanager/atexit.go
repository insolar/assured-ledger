// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"os"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"
)

type ExitContextCallback func() error

type ExitContext struct {
	logger    log.Logger
	callbacks map[string]ExitContextCallback
	once      sync.Once
}

var (
	exitContext     ExitContext
	exitContextOnce sync.Once
)

func InitExitContext(logger log.Logger) {
	initExitContext := func() {
		exitContext.callbacks = make(map[string]ExitContextCallback)
		exitContext.logger = logger
	}

	exitContextOnce.Do(initExitContext)
}

func AtExit(name string, cb ExitContextCallback) {
	exitContext.callbacks[name] = cb
}

func Exit(code int) {
	exit := func() {
		for name, cb := range exitContext.callbacks {
			err := cb()
			if err != nil {
				exitContext.logger.Errorf("Failed to call atexit %s: %s", name, err.Error())
			}
		}
		os.Exit(code)
	}
	exitContext.once.Do(exit)
}
