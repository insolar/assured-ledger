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
