// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package api

import (
	"net/http"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/network"
)

// HealthChecker allows to check network status of a node.
type HealthChecker struct {
	CertificateManager nodeinfo.CertificateManager
	NodeNetwork        network.NodeNetwork
	PulseAccessor      beat.Accessor
}

// NewHealthChecker creates new HealthChecker.
func NewHealthChecker(cm nodeinfo.CertificateManager, nn network.NodeNetwork, pa beat.Accessor) *HealthChecker { // nolint: staticcheck
	return &HealthChecker{CertificateManager: cm, NodeNetwork: nn, PulseAccessor: pa}
}

// CheckHandler is a HTTP handler for health check.
func (hc *HealthChecker) CheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	ctx := r.Context()
	p, err := hc.PulseAccessor.Latest(ctx)
	if err != nil {
		err := errors.W(err, "failed to get latest pulse")
		inslogger.FromContext(ctx).Errorf("[ NodeService.GetStatus ] %s", err.Error())
		_, _ = w.Write([]byte("FAIL"))
		return
	}
	for _, node := range hc.CertificateManager.GetCertificate().GetDiscoveryNodes() {
		if hc.NodeNetwork.GetAccessor(p.PulseNumber).GetWorkingNode(node.GetNodeRef()) == nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("FAIL"))
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
