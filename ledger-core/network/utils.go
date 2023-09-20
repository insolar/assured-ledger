package network

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

// CheckShortIDCollision returns true if nodes contains node with such ShortID
func CheckShortIDCollision(nodes []nodeinfo.NetworkNode, id node.ShortNodeID) bool {
	for _, n := range nodes {
		if id == n.GetNodeID() {
			return true
		}
	}

	return false
}

// ExcludeOrigin returns DiscoveryNode slice without Origin
func ExcludeOrigin(discoveryNodes []nodeinfo.DiscoveryNode, origin reference.Holder) []nodeinfo.DiscoveryNode {
	for i, discoveryNode := range discoveryNodes {
		if discoveryNode.GetNodeRef().Equal(origin) {
			return append(discoveryNodes[:i], discoveryNodes[i+1:]...)
		}
	}
	return discoveryNodes
}

// FindDiscoveryByRef tries to find discovery node in Certificate by reference
func FindDiscoveryByRef(cert nodeinfo.Certificate, ref reference.Holder) nodeinfo.DiscoveryNode {
	bNodes := cert.GetDiscoveryNodes()
	for _, discoveryNode := range bNodes {
		if discoveryNode.GetNodeRef().Equal(ref) {
			return discoveryNode
		}
	}
	return nil
}

func OriginIsDiscovery(cert nodeinfo.Certificate) bool {
	return IsDiscovery(cert.GetNodeRef(), cert)
}

func IsDiscovery(nodeID reference.Holder, cert nodeinfo.Certificate) bool {
	return FindDiscoveryByRef(cert, nodeID) != nil
}

func JoinAssistant(cert nodeinfo.Certificate) nodeinfo.DiscoveryNode {
	bNodes := cert.GetDiscoveryNodes()
	if len(bNodes) == 0 {
		return nil
	}

	sort.Slice(bNodes, func(i, j int) bool {
		a := bNodes[i].GetNodeRef().AsBytes()
		b := bNodes[j].GetNodeRef().AsBytes()
		return bytes.Compare(a, b) > 0
	})
	return bNodes[0]
}

func IsJoinAssistant(nodeID reference.Global, cert nodeinfo.Certificate) bool {
	assist := JoinAssistant(cert)
	if assist == nil {
		return false
	}
	return nodeID.Equal(assist.GetNodeRef())
}

func OriginIsJoinAssistant(cert nodeinfo.Certificate) bool {
	return IsJoinAssistant(cert.GetNodeRef(), cert)
}

func CloseVerbose(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		global.Warnf("[ CloseVerbose ] Failed to close: %s", err.Error())
	}
}

// IsConnectionClosed checks err for connection closed, workaround for poll.ErrNetClosing https://github.com/golang/go/issues/4373
func IsConnectionClosed(err error) bool {
	if err == nil {
		return false
	}
	err = errors.Unwrap(err)
	return err != nil && strings.Contains(err.Error(), "use of closed network connection")
}

// FindDiscoveriesInNodeList returns only discovery nodes from active node list
func FindDiscoveriesInNodeList(nodes []nodeinfo.NetworkNode, cert nodeinfo.Certificate) (result []nodeinfo.NetworkNode) {
	discoveries := map[reference.Global]struct{}{}
	for _, discovery := range cert.GetDiscoveryNodes() {
		discoveries[discovery.GetNodeRef()] = struct{}{}
	}

	for _, n := range nodes {
		if _, ok := discoveries[nodeinfo.NodeRef(n)]; ok {
			result = append(result, n)
		}
	}

	return result
}

func IsClosedPipe(err error) bool {
	if err == nil {
		return false
	}
	err = errors.Unwrap(err)
	return err != nil && strings.Contains(err.Error(), "read/write on closed pipe")
}

func NewPulseContext(ctx context.Context, pulseNumber uint32) context.Context {
	insTraceID := strconv.FormatUint(uint64(pulseNumber), 10) + "_pulse"
	ctx = inslogger.ContextWithTrace(ctx, insTraceID)
	return ctx
}

type CapturingReader struct {
	io.Reader
	buffer bytes.Buffer
}

func NewCapturingReader(reader io.Reader) *CapturingReader {
	return &CapturingReader{Reader: reader}
}

func (r *CapturingReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.buffer.Write(p)
	return n, err
}

func (r *CapturingReader) Captured() []byte {
	return r.buffer.Bytes()
}
