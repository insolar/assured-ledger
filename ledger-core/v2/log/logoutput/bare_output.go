// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package logoutput

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/outputsyslog"
)

type LogOutput uint8

const (
	StdErrOutput LogOutput = iota
	SysLogOutput
)

func (l LogOutput) String() string {
	switch l {
	case StdErrOutput:
		return "stderr"
	case SysLogOutput:
		return "syslog"
	}
	return string(l)
}

func OpenLogBareOutput(output LogOutput, param string) (logcommon.BareOutput, error) {
	switch output {
	case StdErrOutput:
		w := os.Stderr
		return logcommon.BareOutput{
			Writer:         w,
			FlushFn:        w.Sync,
			ProtectedClose: true,
		}, nil
	case SysLogOutput:
		executableName := filepath.Base(os.Args[0])
		w, err := outputsyslog.ConnectSyslogByParam(param, executableName)
		if err != nil {
			return logcommon.BareOutput{}, err
		}
		return logcommon.BareOutput{
			Writer:         w,
			FlushFn:        w.Flush,
			ProtectedClose: false,
		}, nil
	default:
		return logcommon.BareOutput{}, errors.New("unknown output " + output.String())
	}
}
