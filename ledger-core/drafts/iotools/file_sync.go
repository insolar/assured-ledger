// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build !darwin go1.12

package iotools

import "os"

// FileSync calls os.File.Sync with the right parameters.
// This function can be removed once we stop supporting Go 1.11
// on MacOS.
//
// More info: https://golang.org/issue/26650.
func FileSync(f *os.File) error { return f.Sync() }
