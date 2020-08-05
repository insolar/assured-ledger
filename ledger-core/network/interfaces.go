// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package network

import (
	"context"
	"time"

	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/appctl/chorus"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/host"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet/types"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type Report struct {
	PulseData       pulse.Data
	PulseNumber     pulse.Number
	MemberPower     member.Power
	MemberMode      member.OpMode
	IsJoiner        bool
	PopulationValid bool
}

type OnConsensusFinished func(ctx context.Context, report Report)

type BootstrapResult struct {
	Host *host.Host
	// FirstPulseTime    time.Time
	ReconnectRequired bool
	NetworkSize       int
}

// RequestHandler handler function to process incoming requests from network and return responses to these requests.
type RequestHandler func(ctx context.Context, request ReceivedPacket) (response Packet, err error)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/network.HostNetwork -o ../testutils/network -s _mock.go -g

// HostNetwork simple interface to send network requests and process network responses.
type HostNetwork interface {
	component.Starter
	component.Stopper

	// PublicAddress returns public address that can be published for all nodes.
	PublicAddress() string

	// SendRequest send request to a remote node addressed by reference.
	SendRequest(ctx context.Context, t types.PacketType, requestData interface{}, receiver reference.Global) (Future, error)
	// SendRequestToHost send request packet to a remote host.
	SendRequestToHost(ctx context.Context, t types.PacketType, requestData interface{}, receiver *host.Host) (Future, error)
	// RegisterRequestHandler register a handler function to process incoming requests of a specific type.
	// All RegisterRequestHandler calls should be executed before Start.
	RegisterRequestHandler(t types.PacketType, handler RequestHandler)
	// BuildResponse create response to an incoming request with Data set to responseData.
	BuildResponse(ctx context.Context, request Packet, responseData interface{}) Packet
}

// Packet is a packet that is transported via network by HostNetwork.
type Packet interface {
	GetSender() reference.Global
	GetSenderHost() *host.Host
	GetType() types.PacketType
	GetRequest() *packet.Request
	GetResponse() *packet.Response
	GetRequestID() types.RequestID
	String() string
}

type ReceivedPacket interface {
	Packet
	Bytes() []byte
}

// Future allows to handle responses to a previously sent request.
type Future interface {
	Request() Packet
	Response() <-chan ReceivedPacket
	WaitResponse(duration time.Duration) (ReceivedPacket, error)
	Cancel()
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/network.RoutingTable -o ../testutils/network -s _mock.go -g

// RoutingTable contains all routing information of the network.
type RoutingTable interface {
	// Resolve NodeID -> ShortID, Address. Can initiate network requests.
	Resolve(reference.Global) (*host.Host, error)
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/network.Gatewayer -o ../testutils/network -s _mock.go -g

// Gatewayer is a network which can change it's Gateway
type Gatewayer interface {
	Gateway() Gateway
	SwitchState(context.Context, State, pulse.Data)
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/network.Gateway -o ../testutils/network -s _mock.go -g

// Gateway responds for whole network state
type Gateway interface {
	NewGateway(context.Context, State) Gateway

	BeforeRun(context.Context, pulse.Data)
	Run(context.Context, pulse.Data)

	GetState() State

	OnPulseFromConsensus(context.Context, NetworkedPulse)
	OnConsensusFinished(context.Context, Report)

	UpdateState(context.Context, beat.Beat)

	RequestNodeState(chorus.NodeStateFunc)
	CancelNodeState()

	Auther() Auther
	Bootstrapper() Bootstrapper

	EphemeralMode(census.OnlinePopulation) bool

	FailState(ctx context.Context, reason string)
}

type Auther interface {
	// GetCert returns certificate object by node reference, using discovery nodes for signing
	GetCert(context.Context, reference.Global) (nodeinfo.Certificate, error)
	// ValidateCert checks certificate signature
	// TODO make this cert.validate()
	ValidateCert(context.Context, nodeinfo.AuthorizationCertificate) (bool, error)
}

// Bootstrapper interface used to change behavior of handlers in different network states
type Bootstrapper interface {
	HandleNodeAuthorizeRequest(context.Context, Packet) (Packet, error)
	HandleNodeBootstrapRequest(context.Context, Packet) (Packet, error)
	HandleUpdateSchedule(context.Context, Packet) (Packet, error)
	HandleReconnect(context.Context, Packet) (Packet, error)
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/network.Aborter -o ./ -s _mock.go -g

// Aborter provide method for immediately stop node
type Aborter interface {
	// Abort forces to stop all node components
	Abort(ctx context.Context, reason string)
}

type NetworkedPulse = beat.Beat

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/network.TerminationHandler -s _mock.go -g

// TerminationHandler handles such node events as graceful stop, abort, etc.
type TerminationHandler interface {
	// Leave locks until network accept leaving claim
	Leave(context.Context, pulse.Number)
	OnLeaveApproved(context.Context)
	// Terminating is an accessor
	Terminating() bool
}
