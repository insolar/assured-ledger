// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package outputsyslog

import (
	"io"
	"regexp"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
)

type LogLevelWriteCloser interface {
	logcommon.LogLevelWriter
}

// SyslogWriter is an interface matching a syslogcommon.Writer struct.
type SyslogWriteCloser interface {
	io.Closer
	io.Writer
	Debug(m string) error
	Info(m string) error
	Warning(m string) error
	Err(m string) error
	Emerg(m string) error
	Crit(m string) error
}

const DefaultSyslogNetwork = "udp"

var addrRegex = regexp.MustCompile(`^((ip|tcp|udp)(|4|6)|unix|unixgram|unixpacket):`)

func toNetworkAndAddress(s string) (string, string) {
	indexes := addrRegex.FindStringSubmatchIndex(s)
	if len(indexes) == 0 {
		return DefaultSyslogNetwork, s
	}
	return s[:indexes[3]], s[indexes[3]+1:]
}

func ConnectSyslogByParam(outputParam, tag string) (LogLevelWriteCloser, error) {
	if len(outputParam) == 0 || outputParam == "localhost" {
		return ConnectDefaultSyslog(tag)
	}

	nw, addr := toNetworkAndAddress(outputParam)
	return ConnectRemoteSyslog(nw, addr, tag)
}

// TODO PLAT-43 reimplement syslog sender to (1) use []byte for messages (2) support for windows

// SyslogLevelWriter wraps a SyslogWriter and call the right syslog level
// method matching the zerolog level.
func NewSyslogLevelWriter(w SyslogWriteCloser) LogLevelWriteCloser {
	return &syslogWriter{w}
}

type syslogWriter struct {
	w SyslogWriteCloser
}

func (sw *syslogWriter) Flush() error {
	return nil
}

func (sw *syslogWriter) Close() error {
	return sw.w.Close()
}

func (sw *syslogWriter) Write(p []byte) (n int, err error) {
	return sw.w.Write(p)
}

// WriteLevel implements LevelWriter interface.
func (sw *syslogWriter) LogLevelWrite(level logcommon.Level, p []byte) (n int, err error) {
	switch level {
	case logcommon.DebugLevel:
		err = sw.w.Debug(string(p))
	case logcommon.InfoLevel:
		err = sw.w.Info(string(p))
	case logcommon.WarnLevel:
		err = sw.w.Warning(string(p))
	case logcommon.ErrorLevel:
		err = sw.w.Err(string(p))
	case logcommon.FatalLevel:
		err = sw.w.Emerg(string(p))
	case logcommon.PanicLevel:
		err = sw.w.Crit(string(p))
	case logcommon.NoLevel:
		err = sw.w.Info(string(p))
	case logcommon.Disabled:
		// ignore
	default:
		panic("invalid level")
	}
	n = len(p)
	return
}
