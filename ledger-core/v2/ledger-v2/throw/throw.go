// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package throw

type Throwable interface {
	ThrowError()
	ThrowPanic()
}

type E struct {
	Msg   string
	Args  interface{}
	Cause error
}

type W struct {
	Msg  string
	Args interface{}
}

func S(interface{}) Throwable {
	return nil
}
