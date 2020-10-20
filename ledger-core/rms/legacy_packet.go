// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"strconv"

	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet/types"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

type LegacyHost = nwapi.Address

func (p *Packet) SetRequest(request interface{}) {
	var r isRequest_Request
	switch t := request.(type) {
	case *RPCRequest:
		r = &Request_RPC{t}
	case *BootstrapRequest:
		r = &Request_Bootstrap{t}
	case *AuthorizeRequest:
		r = &Request_Authorize{t}
	case *SignCertRequest:
		r = &Request_SignCert{t}
	case *UpdateScheduleRequest:
		r = &Request_UpdateSchedule{t}
	case *ReconnectRequest:
		r = &Request_Reconnect{t}
	default:
		panic("Request payload is not a valid protobuf struct!")
	}
	p.Payload = &Packet_Request{Request: &Request{Request: r}}
}

func (p *Packet) SetResponse(response interface{}) {
	var r isResponse_Response
	switch t := response.(type) {
	case *RPCResponse:
		r = &Response_RPC{t}
	case *BasicResponse:
		r = &Response_Basic{t}
	case *BootstrapResponse:
		r = &Response_Bootstrap{t}
	case *AuthorizeResponse:
		r = &Response_Authorize{t}
	case *SignCertResponse:
		r = &Response_SignCert{t}
	case *ErrorResponse:
		r = &Response_Error{t}
	case *UpdateScheduleResponse:
		r = &Response_UpdateSchedule{t}
	case *ReconnectResponse:
		r = &Response_Reconnect{t}
	default:
		panic("Response payload is not a valid protobuf struct!")
	}
	p.Payload = &Packet_Response{Response: &Response{Response: r}}
}

func (p *Packet) GetType() types.PacketType {
	// TODO: make p.Type of type PacketType instead of uint32
	return types.PacketType(p.Type)
}

func (p *Packet) GetSenderHost() nwapi.Address {
	return p.Sender.Get()
}

func (p *Packet) GetRequestID() types.RequestID {
	return types.RequestID(p.RequestID)
}

func (p *Packet) IsResponse() bool {
	return p.GetResponse() != nil
}

//nolint:goconst
func (p *Packet) DebugString() string {
	if p == nil {
		return "nil"
	}
	return `&Packet{` +
		`Sender:` + p.Sender.Get().String() + `,` +
		`Receiver:` + p.Receiver.Get().String() + `,` +
		`RequestID:` + strconv.FormatUint(p.RequestID, 10) + `,` +
		`TraceID:` + p.TraceID + `,` +
		`Type:` + p.GetType().String() + `,` +
		`IsResponse:` + strconv.FormatBool(p.IsResponse()) + `,` +
		`}`
}
