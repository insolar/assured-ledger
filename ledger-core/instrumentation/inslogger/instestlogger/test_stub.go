// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package instestlogger

import (
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
)

var _ logcommon.TestingLogger = &stubT{}

type stubT struct {}

func (p *stubT) Helper()              {}
func (p *stubT) Log(...interface{})   {}
func (p *stubT) Error(...interface{}) {}
func (p *stubT) Fatal(...interface{}) {}
func (p *stubT) cleanup(bool)         {}
