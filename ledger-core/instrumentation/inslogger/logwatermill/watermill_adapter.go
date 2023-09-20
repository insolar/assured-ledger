package logwatermill

import (
	"github.com/ThreeDotsLabs/watermill"

	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/log/logfmt"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewWatermillLogAdapter(log log.Logger) *WatermillLogAdapter {
	return &WatermillLogAdapter{
		log: log.WithField("service", "watermill"),
	}
}

type WatermillLogAdapter struct {
	log log.Logger
}

func (w *WatermillLogAdapter) event(fields watermill.LogFields, level log.Level, msg string) {
	// don't use w.Debug() etc, value of the "file=..." field would be incorrect
	if fn := w.log.Embeddable().NewEventStruct(level); fn != nil {
		fn(logfmt.LogFields{Msg: msg, Fields: fields}, nil)
	}
}

func (w *WatermillLogAdapter) With(fields watermill.LogFields) watermill.LoggerAdapter {
	l := w.log.WithFields(fields)
	return &WatermillLogAdapter{log: l}
}

func (w *WatermillLogAdapter) Error(msg string, err error, fields watermill.LogFields) {
	w.log.Errorm(throw.E(msg, err), logfmt.FieldMapMarshaller(fields))
}

func (w *WatermillLogAdapter) Info(msg string, fields watermill.LogFields) {
	w.event(fields, log.InfoLevel, msg)
}

func (w *WatermillLogAdapter) Debug(msg string, fields watermill.LogFields) {
	w.event(fields, log.DebugLevel, msg)
}

func (w *WatermillLogAdapter) Trace(msg string, fields watermill.LogFields) {
	w.event(fields, log.DebugLevel, msg)
}
