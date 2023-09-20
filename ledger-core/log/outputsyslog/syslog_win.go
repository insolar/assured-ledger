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
