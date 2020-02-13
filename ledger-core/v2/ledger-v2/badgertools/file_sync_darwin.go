// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build darwin,!go1.12

package badgertools

import (
	"os"
	"syscall"
)

// FileSync calls os.File.Sync with the right parameters.
// This function can be removed once we stop supporting Go 1.11
// on MacOS.
//
// More info: https://golang.org/issue/26650.
func FileSync(f *os.File) error {
	_, _, err := syscall.Syscall(syscall.SYS_FCNTL, f.Fd(), syscall.F_FULLFSYNC, 0)
	if err == 0 {
		return nil
	}
	return err
}
