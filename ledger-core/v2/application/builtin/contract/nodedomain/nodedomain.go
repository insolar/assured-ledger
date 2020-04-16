// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nodedomain

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/appfoundation"
	"github.com/insolar/assured-ledger/ledger-core/v2/application/builtin/proxy/noderecord"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/builtin/foundation"
)

// NodeDomain holds node records.
type NodeDomain struct {
	foundation.BaseContract

	NodeIndexPublicKey foundation.StableMap
}

// NewNodeDomain create new NodeDomain.
func NewNodeDomain() (*NodeDomain, error) {
	return &NodeDomain{
		NodeIndexPublicKey: make(foundation.StableMap),
	}, nil
}

// RegisterNode registers node in system.
func (nd *NodeDomain) RegisterNode(publicKey string, role string) (string, error) {
	root := appfoundation.GetRootMember()
	if *nd.GetContext().Caller != root {
		return "", fmt.Errorf("only root member can register node")
	}

	newNode := noderecord.NewNodeRecord(publicKey, role)
	node, err := newNode.AsChild(nd.GetReference())
	if err != nil {
		return "", fmt.Errorf("failed to save as child: %s", err.Error())
	}

	newNodeRef := node.GetReference().String()
	nd.NodeIndexPublicKey[publicKey] = newNodeRef

	return newNodeRef, err
}

// GetNodeRefByPublicKey returns node reference.
// ins:immutable
func (nd *NodeDomain) GetNodeRefByPublicKey(publicKey string) (string, error) {
	nodeRef, ok := nd.NodeIndexPublicKey[publicKey]
	if !ok {
		return "", fmt.Errorf("network node not found by public key: %s", publicKey)
	}
	return nodeRef, nil
}
