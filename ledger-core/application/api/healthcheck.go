// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package api

import (
	"net/http"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
)

// HealthChecker allows to check network status of a node.
type HealthChecker struct {
	certManager nodeinfo.CertificateManager
	nodeFn      func() beat.NodeSnapshot
}

// NewHealthChecker creates new HealthChecker.
func NewHealthChecker(cm nodeinfo.CertificateManager, nodeFn func() beat.NodeSnapshot) *HealthChecker {
	return &HealthChecker{certManager: cm, nodeFn: nodeFn}
}

// CheckHandler is a HTTP handler for health check.
func (hc *HealthChecker) CheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	ctx := r.Context()
	na := hc.nodeFn()
	if na == nil {
		inslogger.FromContext(ctx).Error("[ NodeService.GetStatus ] failed to get latest pulse")
		_, _ = w.Write([]byte("FAIL"))
		return
	}

	for _, node := range hc.certManager.GetCertificate().GetDiscoveryNodes() {
		if nd := na.FindNodeByRef(node.GetNodeRef()); nd == nil || !nd.IsStateful() {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("FAIL"))
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
