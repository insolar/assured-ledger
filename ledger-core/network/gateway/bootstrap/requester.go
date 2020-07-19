// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package bootstrap

import (
	"bytes"
	"context"
	"math/rand"
	"sort"
	"time"

	"github.com/opentracing/opentracing-go/log"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/adapters"
	"github.com/insolar/assured-ledger/ledger-core/pulse"

	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/instracer"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/host"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet/types"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/network/gateway/bootstrap.Requester -o ./ -s _mock.go -g

type Requester interface {
	Authorize(context.Context, node.Certificate) (*packet.Permit, error)
	Bootstrap(context.Context, *packet.Permit, adapters.Candidate, *pulsestor.Pulse) (*packet.BootstrapResponse, error)
	UpdateSchedule(context.Context, *packet.Permit, pulse.Number) (*packet.UpdateScheduleResponse, error)
	Reconnect(context.Context, *host.Host, *packet.Permit) (*packet.ReconnectResponse, error)
}

func NewRequester(options *network.Options) Requester {
	return &requester{options: options}
}

type requester struct {
	HostNetwork         network.HostNetwork    `inject:""`
	OriginProvider      network.OriginProvider `inject:""` // nolint:staticcheck
	CryptographyService cryptography.Service   `inject:""`

	options *network.Options
}

func (ac *requester) Authorize(ctx context.Context, cert node.Certificate) (*packet.Permit, error) {
	logger := inslogger.FromContext(ctx)

	discoveryNodes := network.ExcludeOrigin(cert.GetDiscoveryNodes(), cert.GetNodeRef())

	if network.OriginIsDiscovery(cert) {
		return ac.authorizeDiscovery(ctx, discoveryNodes, cert)
	}

	rand.Shuffle(
		len(discoveryNodes),
		func(i, j int) {
			discoveryNodes[i], discoveryNodes[j] = discoveryNodes[j], discoveryNodes[i]
		},
	)

	bestResult := &packet.AuthorizeResponse{}

	for _, n := range discoveryNodes {
		h, err := host.NewHostN(n.GetHost(), n.GetNodeRef())
		if err != nil {
			logger.Warnf("Error authorizing to mallformed host %s[%s]: %s",
				n.GetHost(), n.GetNodeRef(), err.Error())
			continue
		}

		logger.Infof("Trying to authorize to node: %s", h.String())
		res, err := ac.authorize(ctx, h, cert)
		if err != nil {
			logger.Warnf("Error authorizing to host %s: %s", h.String(), err.Error())
			continue
		}

		if int(res.DiscoveryCount) < cert.GetMajorityRule() {
			logger.Infof(
				"Check MajorityRule failed on authorize, expect %d, got %d",
				cert.GetMajorityRule(),
				res.DiscoveryCount,
			)

			if res.DiscoveryCount > bestResult.DiscoveryCount {
				bestResult = res
			}

			continue
		}

		return res.Permit, nil
	}

	// todo:  remove best result
	if network.OriginIsDiscovery(cert) && bestResult.Permit != nil {
		return bestResult.Permit, nil
	}

	return nil, throw.New("failed to authorize to any discovery node")
}

func (ac *requester) authorizeDiscovery(ctx context.Context, bNodes []node.DiscoveryNode, cert node.Certificate) (*packet.Permit, error) {
	if len(bNodes) == 0 {
		return nil, throw.Impossible()
	}

	sort.Slice(bNodes, func(i, j int) bool {
		a := bNodes[i].GetNodeRef().AsBytes()
		b := bNodes[j].GetNodeRef().AsBytes()
		return bytes.Compare(a, b) > 0
	})

	logger := inslogger.FromContext(ctx)
	for _, n := range bNodes {
		h, err := host.NewHostN(n.GetHost(), n.GetNodeRef())
		if err != nil {
			logger.Warnf("Error authorizing to mallformed host %s[%s]: %s",
				n.GetHost(), n.GetNodeRef(), err.Error())
			continue
		}

		// panic("asadad")
		logger.Infof("Trying to authorize to node: %s", h.String())
		res, err := ac.authorize(ctx, h, cert)
		if err != nil {
			logger.Warnf("Error authorizing to host %s: %s", h.String(), err.Error())
			continue
		}

		// TODO
		// if int(res.DiscoveryCount) < cert.GetMajorityRule() {
		// 	logger.Infof(
		// 		"Check MajorityRule failed on authorize, expect %d, got %d",
		// 		cert.GetMajorityRule(),
		// 		res.DiscoveryCount,
		// 	)
		//
		// 	if res.DiscoveryCount > bestResult.DiscoveryCount {
		// 		bestResult = res
		// 	}
		//
		// 	continue
		// }

		return res.Permit, nil
	}

	return nil, throw.New("failed to authorize to any discovery node")
}

func (ac *requester) authorize(ctx context.Context, host *host.Host, cert node.AuthorizationCertificate) (*packet.AuthorizeResponse, error) {
	inslogger.FromContext(ctx).Infof("Authorizing on host: %s", host.String())

	ctx, span := instracer.StartSpan(ctx, "AuthorizationController.Authorize")
	span.LogFields(
		log.String("node", host.NodeID.String()),
	)
	defer span.Finish()
	serializedCert, err := mandates.Serialize(cert)
	if err != nil {
		return nil, throw.W(err, "Error serializing certificate")
	}

	authData := &packet.AuthorizeData{Certificate: serializedCert, Version: ac.OriginProvider.GetOrigin().Version()}
	response, err := ac.authorizeWithTimestamp(ctx, host, authData, time.Now().Unix())
	if err != nil {
		return nil, err
	}

	switch response.Code {
	case packet.Success:
		return response, nil
	case packet.WrongMandate:
		return response, throw.New("failed to authorize, wrong mandate")
	case packet.WrongVersion:
		return response, throw.New("failed to authorize, wrong version")
	}

	// retry with received timestamp
	// TODO: change one retry to many
	response, err = ac.authorizeWithTimestamp(ctx, host, authData, response.Timestamp)
	return response, err
}

func (ac *requester) authorizeWithTimestamp(ctx context.Context, h *host.Host, authData *packet.AuthorizeData, timestamp int64) (*packet.AuthorizeResponse, error) {

	authData.Timestamp = timestamp

	data, err := authData.Marshal()
	if err != nil {
		return nil, throw.W(err, "failed to marshal permit")
	}

	signature, err := ac.CryptographyService.Sign(data)
	if err != nil {
		return nil, throw.W(err, "failed to sign permit")
	}

	req := &packet.AuthorizeRequest{AuthorizeData: authData, Signature: signature.Bytes()}

	f, err := ac.HostNetwork.SendRequestToHost(ctx, types.Authorize, req, h)
	if err != nil {
		return nil, throw.W(err, "Error sending Authorize request")
	}
	response, err := f.WaitResponse(ac.options.PacketTimeout)
	if err != nil {
		return nil, throw.W(err, "Error getting response for Authorize request")
	}

	if response.GetResponse().GetError() != nil {
		return nil, throw.New(response.GetResponse().GetError().Error)
	}

	if response.GetResponse() == nil || response.GetResponse().GetAuthorize() == nil {
		return nil, throw.Errorf("Authorize failed: got incorrect response: %s", response)
	}

	return response.GetResponse().GetAuthorize(), nil
}

func (ac *requester) Bootstrap(ctx context.Context, permit *packet.Permit, candidate adapters.Candidate, p *pulsestor.Pulse) (*packet.BootstrapResponse, error) {

	req := &packet.BootstrapRequest{
		CandidateProfile: candidate.Profile(),
		Pulse:            *pulsestor.ToProto(p),
		Permit:           permit,
	}

	f, err := ac.HostNetwork.SendRequestToHost(ctx, types.Bootstrap, req, permit.Payload.ReconnectTo)
	if err != nil {
		return nil, throw.W(err, "Error sending Bootstrap request")
	}

	resp, err := f.WaitResponse(ac.options.PacketTimeout)
	if err != nil {
		return nil, throw.W(err, "Error getting response for Bootstrap request")
	}

	respData := resp.GetResponse().GetBootstrap()
	if respData == nil {
		return nil, throw.New("bad response for bootstrap")
	}

	switch respData.Code {
	case packet.UpdateShortID:
		return respData, throw.New("Bootstrap got UpdateShortID")
	case packet.UpdateSchedule:
		// ac.UpdateSchedule(ctx, permit, p.Number)
		// panic("call bootstrap again")
		return respData, throw.New("Bootstrap got UpdateSchedule")
	case packet.Reject:
		return respData, throw.New("Bootstrap request rejected")
	case packet.Retry:
		// todo sleep and retry ac.Bootstrap()
		// panic("packet.Retry")
		time.Sleep(time.Second)
		return ac.Bootstrap(ctx, permit, candidate, p)
	}

	// case Accepted
	return respData, nil

}

func (ac *requester) UpdateSchedule(ctx context.Context, permit *packet.Permit, pulse pulse.Number) (*packet.UpdateScheduleResponse, error) {

	req := &packet.UpdateScheduleRequest{
		LastNodePulse: pulse,
		Permit:        permit,
	}

	f, err := ac.HostNetwork.SendRequestToHost(ctx, types.UpdateSchedule, req, permit.Payload.ReconnectTo)
	if err != nil {
		return nil, throw.W(err, "Error sending UpdateSchedule request")
	}

	resp, err := f.WaitResponse(ac.options.PacketTimeout)
	if err != nil {
		return nil, throw.W(err, "Error getting response for UpdateSchedule request")
	}

	return resp.GetResponse().GetUpdateSchedule(), nil
}

func (ac *requester) Reconnect(ctx context.Context, h *host.Host, permit *packet.Permit) (*packet.ReconnectResponse, error) {
	req := &packet.ReconnectRequest{
		ReconnectTo: *permit.Payload.ReconnectTo,
		Permit:      permit,
	}

	f, err := ac.HostNetwork.SendRequestToHost(ctx, types.Reconnect, req, h)
	if err != nil {
		return nil, throw.W(err, "Error sending Reconnect request")
	}

	resp, err := f.WaitResponse(ac.options.PacketTimeout)
	if err != nil {
		return nil, throw.W(err, "Error getting response for Reconnect request")
	}

	return resp.GetResponse().GetReconnect(), nil
}
