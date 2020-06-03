// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package inslogger

import (
	"fmt"
	"net/http"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/global"
)

// ServeHTTP is an HTTP handler that changes the global minimum log level
func NewLoglevelChangeHandler() http.Handler {
	handler := &loglevelChangeHandler{}
	return handler
}

type loglevelChangeHandler struct {
}

func (h *loglevelChangeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	levelStr := "(nil)"
	if values["level"] != nil {
		levelStr = values["level"][0]
	}
	level, err := log.ParseLevel(levelStr)
	if err != nil {
		w.WriteHeader(500)
		_, _ = fmt.Fprintf(w, "Invalid level '%v': %v\n", levelStr, err)
		return
	}

	err = global.SetFilter(level)

	if err == nil {
		w.WriteHeader(200)
		_, _ = fmt.Fprintf(w, "New log level: '%v'\n", levelStr)
		return
	}

	w.WriteHeader(500)
	_, _ = fmt.Fprintf(w, "Logger doesn't support global log level(s): %v\n", err)
}
