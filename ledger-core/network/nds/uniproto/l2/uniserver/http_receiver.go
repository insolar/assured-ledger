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
