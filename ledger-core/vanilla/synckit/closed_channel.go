// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package synckit

import "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

type SignalChannel = <-chan struct{}

type ClosableSignalChannel = chan struct{}

func ClosedChannel() SignalChannel {
	return closedChan
}

var closedChan = func() SignalChannel {
	c := make(ClosableSignalChannel)
	close(c)
	return c
}()

func SafeClose(c ClosableSignalChannel) (err error) {
	defer func() {
		err = throw.R(recover(), err)
	}()
	close(c)
	return nil
}
