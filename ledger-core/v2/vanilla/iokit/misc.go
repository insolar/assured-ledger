// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package iokit

import "io"

func SafeCloseWithDefault(c io.Closer, nilErrorFn func() error) error {
	switch {
	case c != nil:
		return c.Close()
	case nilErrorFn == nil:
		return nil
	default:
		return nilErrorFn()
	}
}

func SafeClose(c io.Closer) error {
	return SafeCloseWithDefault(c, nil)
}

func SafeCloseChain(c io.Closer, prev error) error {
	if c != nil {
		err := c.Close()
		if prev == nil {
			return err
		}
	}
	return prev
}
