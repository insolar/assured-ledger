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
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
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

var BareStdErr = logcommon.BareOutput{
	Writer:         os.Stderr,
	FlushFn:        os.Stderr.Sync,
	ProtectedClose: true,
}

func OpenLogBareOutput(output LogOutput, param string) (logcommon.BareOutput, error) {
	switch output {
	case StdErrOutput:
		o := BareStdErr
		if o.Writer == nil {
			return o, throw.IllegalState()
		}
		return o, nil
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
