// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build !windows

package iotools

import (
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Mmap uses the mmap system call to memory-map a file. If writable is true,
// memory protection of the pages is set so that they may be written to as well.
func Mmap(fd *os.File, writable bool, size int64) ([]byte, error) {
	mtype := unix.PROT_READ
	if writable {
		mtype |= unix.PROT_WRITE
	}
	return unix.Mmap(int(fd.Fd()), 0, int(size), mtype, unix.MAP_SHARED)
}

// Munmap unmaps a previously mapped slice.
func Munmap(b []byte) error {
	return unix.Munmap(b)
}

// Madvise uses the madvise system call to give advise about the use of memory
// when using a slice that is memory-mapped to a file. Set the readahead flag to
// false if page references are expected in random order.
func Madvise(b []byte, readahead bool) error {
	flags := unix.MADV_NORMAL
	if !readahead {
		flags = unix.MADV_RANDOM
	}
	return madvise(b, flags)
}

// This is required because the unix package does not support the madvise system call on OS X.
func madvise(b []byte, advice int) (err error) {
	_, _, e1 := syscall.Syscall(syscall.SYS_MADVISE, uintptr(unsafe.Pointer(&b[0])),
		uintptr(len(b)), uintptr(advice))
	if e1 != 0 {
		err = e1
	}
	return
}
