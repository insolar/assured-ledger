// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package network

import (
	"bytes"
	"context"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/global"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/node"
)

func WaitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return true // completed normally
	case <-time.After(timeout):
		return false // timed out
	}
}

// CheckShortIDCollision returns true if nodes contains node with such ShortID
func CheckShortIDCollision(nodes []insolar.NetworkNode, id insolar.ShortNodeID) bool {
	for _, n := range nodes {
		if id == n.ShortID() {
			return true
		}
	}

	return false
}

// GenerateUniqueShortID correct ShortID of the node so it does not conflict with existing active node list
func GenerateUniqueShortID(nodes []insolar.NetworkNode, nodeID insolar.Reference) insolar.ShortNodeID {
	shortID := insolar.ShortNodeID(node.GenerateUintShortID(nodeID))
	if !CheckShortIDCollision(nodes, shortID) {
		return shortID
	}
	return regenerateShortID(nodes, shortID)
}

func regenerateShortID(nodes []insolar.NetworkNode, shortID insolar.ShortNodeID) insolar.ShortNodeID {
	shortIDs := make([]insolar.ShortNodeID, len(nodes))
	for i, activeNode := range nodes {
		shortIDs[i] = activeNode.ShortID()
	}
	sort.Slice(shortIDs, func(i, j int) bool {
		return shortIDs[i] < shortIDs[j]
	})
	return generateNonConflictingID(shortIDs, shortID)
}

func generateNonConflictingID(sortedSlice []insolar.ShortNodeID, conflictingID insolar.ShortNodeID) insolar.ShortNodeID {
	index := sort.Search(len(sortedSlice), func(i int) bool {
		return sortedSlice[i] >= conflictingID
	})
	result := conflictingID
	repeated := false
	for {
		if result == math.MaxUint32 {
			if !repeated {
				repeated = true
				result = 0
				index = 0
			} else {
				panic("[ generateNonConflictingID ] shortID overflow twice")
			}
		}
		index++
		result++
		if index >= len(sortedSlice) || result != sortedSlice[index] {
			return result
		}
	}
}

// ExcludeOrigin returns DiscoveryNode slice without Origin
func ExcludeOrigin(discoveryNodes []insolar.DiscoveryNode, origin insolar.Reference) []insolar.DiscoveryNode {
	for i, discoveryNode := range discoveryNodes {
		if origin.Equal(discoveryNode.GetNodeRef()) {
			return append(discoveryNodes[:i], discoveryNodes[i+1:]...)
		}
	}
	return discoveryNodes
}

// FindDiscoveryByRef tries to find discovery node in Certificate by reference
func FindDiscoveryByRef(cert insolar.Certificate, ref insolar.Reference) insolar.DiscoveryNode {
	bNodes := cert.GetDiscoveryNodes()
	for _, discoveryNode := range bNodes {
		if ref.Equal(discoveryNode.GetNodeRef()) {
			return discoveryNode
		}
	}
	return nil
}

func OriginIsDiscovery(cert insolar.Certificate) bool {
	return IsDiscovery(cert.GetNodeRef(), cert)
}

func IsDiscovery(nodeID insolar.Reference, cert insolar.Certificate) bool {
	return FindDiscoveryByRef(cert, nodeID) != nil
}

func JoinAssistant(cert insolar.Certificate) insolar.DiscoveryNode {
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

func IsJoinAssistant(nodeID insolar.Reference, cert insolar.Certificate) bool {
	assist := JoinAssistant(cert)
	if assist == nil {
		return false
	}
	return nodeID.Equal(assist.GetNodeRef())
}

func OriginIsJoinAssistant(cert insolar.Certificate) bool {
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
	err = errors.Cause(err)
	return strings.Contains(err.Error(), "use of closed network connection")
}

// FindDiscoveriesInNodeList returns only discovery nodes from active node list
func FindDiscoveriesInNodeList(nodes []insolar.NetworkNode, cert insolar.Certificate) []insolar.NetworkNode {
	discovery := cert.GetDiscoveryNodes()
	result := make([]insolar.NetworkNode, 0)

	for _, d := range discovery {
		for _, n := range nodes {
			if d.GetNodeRef().Equal(n.ID()) {
				result = append(result, n)
				break
			}
		}
	}

	return result
}

func IsClosedPipe(err error) bool {
	if err == nil {
		return false
	}
	err = errors.Cause(err)
	return strings.Contains(err.Error(), "read/write on closed pipe")
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
