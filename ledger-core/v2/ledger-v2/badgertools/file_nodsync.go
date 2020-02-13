// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build dragonfly freebsd windows

package badgertools

import "syscall"

func init() {
	datasyncFileFlag = syscall.O_SYNC
}
