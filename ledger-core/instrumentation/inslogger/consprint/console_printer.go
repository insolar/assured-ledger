package consprint

import (
	"errors"
	"io"
	"os"

	"github.com/rs/zerolog"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

// prevents leak of explicit zerolog dependency
type Config = zerolog.ConsoleWriter

func NewConsolePrinter(w io.Writer, config Config) io.Writer {
	if w == nil {
		panic(throw.IllegalValue())
	}

	if config.PartsOrder == nil {
		config.PartsOrder = []string{
			zerolog.TimestampFieldName,
			zerolog.LevelFieldName,
			zerolog.MessageFieldName,
			zerolog.CallerFieldName,
		}
	}

	config.Out = w
	return &closableConsoleWriter{config}
}

var _ io.WriteCloser = &closableConsoleWriter{}

type closableConsoleWriter struct {
	zerolog.ConsoleWriter
}

func (p *closableConsoleWriter) Close() error {
	if c, ok := p.Out.(io.Closer); ok {
		return c.Close()
	}
	return errors.New("unsupported: Close")
}

func (p *closableConsoleWriter) Sync() error {
	if c, ok := p.Out.(*os.File); ok {
		return c.Sync()
	}
	return errors.New("unsupported: Sync")
}
