// +build dragonfly freebsd windows

package iotools

import "syscall"

func init() {
	datasyncFileFlag = syscall.O_SYNC
}
