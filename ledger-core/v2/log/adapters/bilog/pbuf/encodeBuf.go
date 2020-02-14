// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pbuf

import "io"

var _ io.Writer = &encodeBuf{}
var _ io.ByteWriter = &encodeBuf{}

type encodeBuf struct {
	dst []byte
}

func (e *encodeBuf) WriteByte(c byte) error {
	e.dst = append(e.dst, c)
	return nil
}

func (e *encodeBuf) Write(p []byte) (n int, err error) {
	e.dst = append(e.dst, p...)
	return len(p), nil
}
