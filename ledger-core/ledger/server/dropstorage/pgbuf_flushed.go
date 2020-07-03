// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package dropstorage

import (
	"io"
)

var _ io.WriterTo = &FlushedPages{}

type FlushedPages struct {
	first *bufferPage
}

func (p *FlushedPages) IsEmpty() bool {
	return p.first == nil
}

func (p *FlushedPages) WriteTo(w io.Writer) (n int64, err error) {
	x := 0
	next := p.first

	for next != nil {
		next.wait.Wait()

		b := next.buffer[:next.used.Load()]

		x, err = w.Write(b)
		n += int64(x)
		if err != nil {
			return
		}
		next = next.next
	}
	return
}

