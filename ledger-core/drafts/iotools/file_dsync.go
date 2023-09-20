// +build !dragonfly,!freebsd,!windows

package iotools

import "golang.org/x/sys/unix"

func init() {
	datasyncFileFlag = unix.O_DSYNC
}
