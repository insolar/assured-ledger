package utils

import (
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/log/logfmt"
)

type LogInterceptHandler = func(interface{})

func InterceptLog(logger log.Logger, interceptFn LogInterceptHandler) log.Logger {
	if interceptFn == nil {
		return logger
	}

	embedded := logger.Embeddable()
	li := logInterceptor{embedded, interceptFn}
	if _, ok := embedded.(logcommon.EmbeddedLoggerOptional); ok {
		return log.WrapEmbeddedLogger(logInterceptorWithFields{li})
	}
	return log.WrapEmbeddedLogger(li)
}

type logInterceptor struct {
	logcommon.EmbeddedLogger
	interceptFn LogInterceptHandler
}

func (lc logInterceptor) NewEventStruct(level logcommon.Level) func(i interface{}, marshallers []logfmt.LogFieldMarshaller) {
	return func(i interface{}, marshallers []logfmt.LogFieldMarshaller) {
		lc.EmbeddedLogger.NewEventStruct(level)(i, marshallers)
		lc.interceptFn(i)
	}
}

func (lc logInterceptor) NewEvent(level logcommon.Level) func(args []interface{}) {
	return func(args []interface{}) {
		lc.EmbeddedLogger.NewEvent(level)(args)
		if len(args) > 0 {
			lc.interceptFn(args[0])
		}
	}
}

var _ logcommon.EmbeddedLoggerOptional = logInterceptorWithFields{}

type logInterceptorWithFields struct {
	logInterceptor
}

func (lc logInterceptorWithFields) WithFields(fields map[string]interface{}) logcommon.EmbeddedLogger {
	return logInterceptorWithFields{logInterceptor{lc.EmbeddedLogger.(logcommon.EmbeddedLoggerOptional).WithFields(fields), lc.interceptFn}}
}

func (lc logInterceptorWithFields) WithField(name string, value interface{}) logcommon.EmbeddedLogger {
	return logInterceptorWithFields{logInterceptor{lc.EmbeddedLogger.(logcommon.EmbeddedLoggerOptional).WithField(name, value), lc.interceptFn}}
}
