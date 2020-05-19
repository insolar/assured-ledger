// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package api

import (
	"context"
	"net/http"
	"strconv"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"

	network2 "github.com/insolar/assured-ledger/ledger-core/v2/network"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/network"

	"github.com/stretchr/testify/assert"
)

type mockResponseWriter struct {
	header http.Header
	body   *[]byte
}

func newMockResponseWriter() mockResponseWriter {
	return mockResponseWriter{
		header: make(http.Header, 0),
		body:   new([]byte),
	}
}

func (w mockResponseWriter) Header() http.Header {
	return w.header
}

func (w mockResponseWriter) Write(data []byte) (int, error) {
	*w.body = append(*w.body, data...)
	return len(data), nil
}

func (w mockResponseWriter) WriteHeader(statusCode int) {
	w.header["status"] = []string{strconv.Itoa(statusCode)}
}

func randomNodeList(t *testing.T, size int) []node.DiscoveryNode {
	list := make([]node.DiscoveryNode, size)
	for i := 0; i < size; i++ {
		dn := testutils.NewDiscoveryNodeMock(t)
		r := gen.Reference()
		dn.GetNodeRefMock.Set(func() reference.Global {
			return r
		})
		list[i] = dn
	}
	return list
}

func mockCertManager(t *testing.T, nodeList []node.DiscoveryNode) *testutils.CertificateManagerMock {
	cm := testutils.NewCertificateManagerMock(t)
	cm.GetCertificateMock.Set(func() node.Certificate {
		c := testutils.NewCertificateMock(t)
		c.GetDiscoveryNodesMock.Set(func() []node.DiscoveryNode {
			return nodeList
		})
		return c
	})
	return cm
}

func mockNodeNetwork(t *testing.T, nodeList []node.DiscoveryNode) *network.NodeNetworkMock {
	nn := network.NewNodeNetworkMock(t)
	nodeMap := make(map[reference.Global]node.DiscoveryNode)
	for _, node := range nodeList {
		nodeMap[node.GetNodeRef()] = node
	}

	accessorMock := network.NewAccessorMock(t)
	accessorMock.GetWorkingNodeMock.Set(func(ref reference.Global) node.NetworkNode {
		if _, ok := nodeMap[ref]; ok {
			return network.NewNetworkNodeMock(t)
		}
		return nil
	})

	nn.GetAccessorMock.Set(func(p1 pulse.Number) network2.Accessor {
		return accessorMock
	})

	return nn
}

func mockPulseAccessor(t *testing.T) *pulsestor.AccessorMock {
	pa := pulsestor.NewAccessorMock(t)
	pa.LatestMock.Set(func(context.Context) (pulsestor.Pulse, error) {
		return *pulsestor.GenesisPulse, nil
	})
	return pa
}

func TestHealthChecker_CheckHandler(t *testing.T) {
	tests := []struct {
		name         string
		from, to     int
		status, body string
	}{
		{"all discovery", 0, 20, "200", "OK"},
		{"not enough discovery", 0, 11, "500", "FAIL"},
		{"extra nodes", 0, 40, "200", "OK"},
		{"not enough discovery and extra nodes", 5, 40, "500", "FAIL"},
	}
	nodes := randomNodeList(t, 40)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hc := NewHealthChecker(
				mockCertManager(t, nodes[:20]),
				mockNodeNetwork(t, nodes[test.from:test.to]),
				mockPulseAccessor(t),
			)
			w := newMockResponseWriter()
			hc.CheckHandler(w, new(http.Request))

			assert.Equal(t, w.header["status"], []string{test.status})
			assert.Equal(t, *w.body, []byte(test.body))
		})
	}
}
