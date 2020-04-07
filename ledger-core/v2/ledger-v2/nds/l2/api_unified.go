// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l2

import (
	"io"
	"net"
)

type UnifiedPeer interface {
	UnifiedPeer()
	//VerifyAddress(remote net.Addr) (l1.Address, apinetwork.VerifyHeaderFunc, error)
	// Set PK
	// Set NodeId

}

type StreamReadyFunc func(io.ReadCloser, PayloadLengthLimit)
type UnifiedDataReceiver interface {
	ReceiveStream(local, remote net.Addr, r io.ReadCloser, limit PayloadLengthLimit, readyFn StreamReadyFunc)
	ReceiveDatagram(local, remote net.Addr, b []byte)
	ReceiveError(local, remote net.Addr, err error)
}

type PayloadLengthLimit uint8

const (
	_ PayloadLengthLimit = iota
	DetectByFirstPayloadLength
	NonExcessivePayloadLength
	UnlimitedPayloadLength
)
