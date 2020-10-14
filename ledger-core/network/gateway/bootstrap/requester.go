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

	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/version"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/adapters"
	"github.com/insolar/assured-ledger/ledger-core/pulse"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet/types"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/rms/legacyhost"
)

const bootstrapRetryCount = 2

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/network/gateway/bootstrap.Requester -o ./ -s _mock.go -g

type Requester interface {
	Authorize(context.Context, nodeinfo.Certificate) (*rms.Permit, error)
	Bootstrap(context.Context, *rms.Permit, adapters.Candidate) (*rms.BootstrapResponse, error)
	UpdateSchedule(context.Context, *rms.Permit, pulse.Number) (*rms.UpdateScheduleResponse, error)
	Reconnect(context.Context, *legacyhost.Host, *rms.Permit) (*rms.ReconnectResponse, error)
}

func NewRequester(options *network.Options) Requester {
	return &requester{options: options}
}

type requester struct {
	HostNetwork         network.HostNetwork  `inject:""`
	CryptographyService cryptography.Service `inject:""`

	options *network.Options
	retry   int
}

func (ac *requester) Authorize(ctx context.Context, cert nodeinfo.Certificate) (*rms.Permit, error) {
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

	bestResult := &rms.AuthorizeResponse{}

	for _, n := range discoveryNodes {
		h := nwapi.NewHostPort(n.GetHost(), false)

		logger.Infof("Trying to authorize to node: %s", h.String())
		res, err := ac.authorize(ctx, &h, cert)
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

func (ac *requester) authorizeDiscovery(ctx context.Context, nodes []nodeinfo.DiscoveryNode, cert nodeinfo.AuthorizationCertificate) (*rms.Permit, error) {
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
		h := nwapi.NewHostPort(n.GetHost(), false)

		logger.Infof("Trying to authorize to node: %s", h.String())
		res, err := ac.authorize(ctx, &h, cert)
		if err != nil {
			logger.Warnf("Error authorizing to host %s: %s", h.String(), err.Error())
			continue
		}

		return res.Permit, nil
	}

	return nil, throw.New("failed to authorize to any discovery node")
}

func (ac *requester) authorize(ctx context.Context, address *nwapi.Address, cert nodeinfo.AuthorizationCertificate) (*rms.AuthorizeResponse, error) {
	inslogger.FromContext(ctx).Infof("Authorizing on address: %s", address.String())

	// ctx, span := instracer.StartSpan(ctx, "AuthorizationController.Authorize")
	// span.LogFields(
	// 	log.String("node", host.NodeID.String()),
	// )
	// defer span.Finish()
	serializedCert, err := mandates.Serialize(cert)
	if err != nil {
		return nil, throw.W(err, "Error serializing certificate")
	}

	authData := &rms.AuthorizeData{Certificate: serializedCert, Version: version.Version}
	response, err := ac.authorizeWithTimestamp(ctx, address, authData, time.Now().Unix())
	if err != nil {
		return nil, err
	}

	switch response.Code {
	case rms.AuthorizeResponseCode_Success:
		return response, nil
	case rms.AuthorizeResponseCode_WrongMandate:
		return response, throw.New("failed to authorize, wrong mandate")
	case rms.AuthorizeResponseCode_WrongVersion:
		return response, throw.New("failed to authorize, wrong version")
	}

	// retry with received timestamp
	// TODO: change one retry to many
	response, err = ac.authorizeWithTimestamp(ctx, address, authData, response.Timestamp)
	return response, err
}

func (ac *requester) authorizeWithTimestamp(ctx context.Context, h *nwapi.Address, authData *rms.AuthorizeData, timestamp int64) (*rms.AuthorizeResponse, error) {

	authData.Timestamp = timestamp

	data, err := authData.Marshal()
	if err != nil {
		return nil, throw.W(err, "failed to marshal permit")
	}

	signature, err := ac.CryptographyService.Sign(data)
	if err != nil {
		return nil, throw.W(err, "failed to sign permit")
	}

	req := &rms.AuthorizeRequest{AuthorizeData: authData, Signature: signature.Bytes()}

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

func (ac *requester) Bootstrap(ctx context.Context, permit *rms.Permit, candidate adapters.Candidate) (*rms.BootstrapResponse, error) {

	req := &rms.BootstrapRequest{
		CandidateProfile: candidate.Profile(),
		Permit:           permit,
	}

	f, err := ac.HostNetwork.SendRequestToHost(ctx, types.Bootstrap, req, &permit.Payload.ReconnectTo)
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
	case rms.BootstrapResponseCode_UpdateShortID:
		return respData, throw.New("Bootstrap got UpdateShortID")
	case rms.BootstrapResponseCode_UpdateSchedule:
		// ac.UpdateSchedule(ctx, permit, p.Number)
		// panic("call bootstrap again")
		return respData, throw.New("Bootstrap got UpdateSchedule")
	case rms.BootstrapResponseCode_Reject:
		return respData, throw.New("Bootstrap request rejected")
	case rms.BootstrapResponseCode_Retry:
		time.Sleep(time.Second * 2)
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

func (ac *requester) UpdateSchedule(ctx context.Context, permit *rms.Permit, pulse pulse.Number) (*rms.UpdateScheduleResponse, error) {

	req := &rms.UpdateScheduleRequest{
		LastNodePulse: pulse,
		Permit:        permit,
	}

	f, err := ac.HostNetwork.SendRequestToHost(ctx, types.UpdateSchedule, req, &permit.Payload.ReconnectTo)
	if err != nil {
		return nil, throw.W(err, "Error sending UpdateSchedule request")
	}

	resp, err := f.WaitResponse(ac.options.PacketTimeout)
	if err != nil {
		return nil, throw.W(err, "Error getting response for UpdateSchedule request")
	}

	return resp.GetResponse().GetUpdateSchedule(), nil
}

func (ac *requester) Reconnect(ctx context.Context, h *legacyhost.Host, permit *rms.Permit) (*rms.ReconnectResponse, error) {
	req := &rms.ReconnectRequest{
		ReconnectTo: permit.Payload.ReconnectTo,
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
