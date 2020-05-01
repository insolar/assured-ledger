// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

var _ Serializable = Interceptor{}

type Interceptor struct {
	Delegate    Serializable
	InterceptFn func([]byte) error
}

func (v Interceptor) ProtoSize() int {
	return v.Delegate.ProtoSize()
}

func (v Interceptor) MarshalTo(b []byte) (n int, err error) {
	n, err = v.Delegate.MarshalTo(b)
	if err == nil {
		err = v.InterceptFn(b[:n])
	}
	return n, err
}

func (v Interceptor) Unmarshal(b []byte) error {
	err := v.Delegate.Unmarshal(b)
	if err == nil {
		err = v.InterceptFn(b)
	}
	return err
}
