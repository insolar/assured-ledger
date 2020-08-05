// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package api

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/network/mutable"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/testutils"
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

func randomNodeList(t *testing.T, size int) []nodeinfo.DiscoveryNode {
	list := make([]nodeinfo.DiscoveryNode, size)
	for i := 0; i < size; i++ {
		dn := testutils.NewDiscoveryNodeMock(t)
		r := gen.UniqueGlobalRef()
		dn.GetNodeRefMock.Set(func() reference.Global {
			return r
		})
		list[i] = dn
	}
	return list
}

func mockCertManager(t *testing.T, nodeList []nodeinfo.DiscoveryNode) *testutils.CertificateManagerMock {
	cm := testutils.NewCertificateManagerMock(t)
	cm.GetCertificateMock.Set(func() nodeinfo.Certificate {
		c := testutils.NewCertificateMock(t)
		c.GetDiscoveryNodesMock.Set(func() []nodeinfo.DiscoveryNode {
			return nodeList
		})
		return c
	})
	return cm
}

func mockNodeNetwork(t *testing.T, nodeList []nodeinfo.DiscoveryNode) *beat.NodeNetworkMock {
	nn := beat.NewNodeNetworkMock(t)
	nodeMap := make(map[reference.Global]nodeinfo.DiscoveryNode)
	for _, node := range nodeList {
		nodeMap[node.GetNodeRef()] = node
	}

	accessorMock := beat.NewNodeSnapshotMock(t)
	accessorMock.FindNodeByRefMock.Set(func(ref reference.Global) nodeinfo.NetworkNode {
		if _, ok := nodeMap[ref]; ok {
			return mutable.NewTestNode(ref, member.PrimaryRoleNeutral, "")
		}
		return nil
	})

	nn.GetNodeSnapshotMock.Return(accessorMock)
	nn.FindAnyLatestNodeSnapshotMock.Return(accessorMock)

	return nn
}

func mockPulseAccessor(t *testing.T) *beat.AppenderMock {
	pa := beat.NewAppenderMock(t)
	pa.LatestTimeBeatMock.Return(pulsestor.GenesisPulse, nil)
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
			)
			w := newMockResponseWriter()
			hc.CheckHandler(w, new(http.Request))

			assert.Equal(t, w.header["status"], []string{test.status})
			assert.Equal(t, *w.body, []byte(test.body))
		})
	}
}
