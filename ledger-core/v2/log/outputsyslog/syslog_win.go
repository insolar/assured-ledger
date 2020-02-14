// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build windows

package outputsyslog

import (
	"errors"
)

func ConnectDefaultSyslog(tag string) (LogLevelWriteCloser, error) {
	return nil, errors.New("not implemented for Windows")
}

func ConnectRemoteSyslog(network, raddr string, tag string) (LogLevelWriteCloser, error) {
	return nil, errors.New("not implemented for Windows")
}
