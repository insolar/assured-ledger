// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniserver

import (
	"bufio"
	"net/http"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
)

type HTTPReceiver struct {
	PeerManager *PeerManager
	Relayer     Relayer
	Parser      uniproto.Parser
	VerifyFn    uniproto.VerifyHeaderFunc
	Format      TransportStreamFormat
}

type HTTPReceiverFunc = func(*http.Request, *bufio.Writer, *HTTPReceiver) error
