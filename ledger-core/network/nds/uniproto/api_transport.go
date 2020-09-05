// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniproto

import (
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l1"
)

// OutFunc receives a specific transport and should use its methods to send data.
// WARNING! When Sessionless transport is allowed then this func MUST ONLY send a single chunk of data. Otherwise delivery will be broken.
// NB! Should use l1.BasicOutTransport.Send when source of data is file or another network connection.
// Function should return (canRetry) when retry is possible for this transfer.
type OutFunc func(l1.BasicOutTransport) (canRetry bool, err error)

type OutTransport interface {
	// UseSessionless applies OutFunc on a sessionless transport. This function is non-blocking when a connection is established.
	// WARNING! OutFunc MUST ONLY send a single chunk of data. Otherwise delivery will be broken.
	UseSessionless(OutFunc) error
	// UseSessionful applies OutFunc on a sessionful transport, either for small or for large packets.
	// This function is blocking. Both small and large transports have individual mutexes.
	UseSessionful(size int64, applyFn OutFunc) error
	// CanUseSessionless returns true when Sessionless server is listening (otherwise replies will be lost) and size is supported by sessionless transport.
	CanUseSessionless(size int64) bool
	// EnsureConnect forces connection of sessionful transport for small packet. Is used to check availability of a peer.
	// This function is non-blocking when a connection is established.
	EnsureConnect() error
}
