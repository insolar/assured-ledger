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

	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/version"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/adapters"
	"github.com/insolar/assured-ledger/ledger-core/pulse"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/instracer"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/host"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet/types"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
)

const bootstrapRetryCount = 3

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/network/gateway/bootstrap.Requester -o ./ -s _mock.go -g

type Requester interface {
	Authorize(context.Context, nodeinfo.Certificate) (*packet.Permit, error)
	Bootstrap(context.Context, *packet.Permit, adapters.Candidate) (*packet.BootstrapResponse, error)
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
	retry   int
}

func (ac *requester) Authorize(ctx context.Context, cert nodeinfo.Certificate) (*packet.Permit, error) {
	logger := inslogger.FromContext(ctx)

	discoveryNodes := network.ExcludeOrigin(cert.GetDiscoveryNodes(), cert.GetNodeRef())

	if network.IsDiscoveryCert(cert) {
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
	if network.IsDiscoveryCert(cert) && bestResult.Permit != nil {
		return bestResult.Permit, nil
	}

	return nil, throw.New("failed to authorize to any discovery node")
}

func (ac *requester) authorizeDiscovery(ctx context.Context, nodes []nodeinfo.DiscoveryNode, cert nodeinfo.AuthorizationCertificate) (*packet.Permit, error) {
	if len(nodes) == 0 {
		return nil, throw.Impossible()
	}

	sort.Slice(nodes, func(i, j int) bool {
		a := nodes[i].GetNodeRef().AsBytes()
		b := nodes[j].GetNodeRef().AsBytes()
		return bytes.Compare(a, b) > 0
	})

	logger := inslogger.FromContext(ctx)
	for _, n := range nodes {
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

		return res.Permit, nil
	}

	return nil, throw.New("failed to authorize to any discovery node")
}

func (ac *requester) authorize(ctx context.Context, host *host.Host, cert nodeinfo.AuthorizationCertificate) (*packet.AuthorizeResponse, error) {
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

	authData := &packet.AuthorizeData{Certificate: serializedCert, Version: version.Version}
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

func (ac *requester) Bootstrap(ctx context.Context, permit *packet.Permit, candidate adapters.Candidate) (*packet.BootstrapResponse, error) {

	req := &packet.BootstrapRequest{
		CandidateProfile: candidate.Profile(),
		Permit:           permit,
	}

	backoff := synckit.Backoff{
		Min:    ac.options.MinTimeout,
		Max:    ac.options.MaxTimeout,
		Factor: float64(ac.options.TimeoutMult),
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
		time.Sleep(backoff.Duration())
		ac.retry++
		if ac.retry > bootstrapRetryCount {
			ac.retry = 0
			return respData, throw.New("Retry bootstrap failed")
		}
		return ac.Bootstrap(ctx, permit, candidate)
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
