// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nodestorage

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/insolar/nodestorage.Accessor -o ./ -s _mock.go -g

// Accessor provides info about active nodes.
type Accessor interface {
	All(pulse pulse.Number) ([]node.Node, error)
	InRole(pulse pulse.Number, role node.StaticRole) ([]node.Node, error)
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/insolar/nodestorage.Modifier -o ./ -s _mock.go -g

// Modifier provides methods for setting active nodes.
type Modifier interface {
	Set(pulse pulse.Number, nodes []node.Node) error
	DeleteForPN(pulse pulse.Number)
}

// Storage is an in-memory active node storage for each pulse. It's required to calculate node roles
// for past pulses to locate data.
// It should only contain previous N pulses. It should be stored on disk.
type Storage struct {
	lock  sync.RWMutex
	nodes map[pulse.Number][]node.Node
}

// NewStorage create new instance of Storage
func NewStorage() *Storage {
	// return new(nodeStorage)
	return &Storage{nodes: map[pulse.Number][]node.Node{}}
}

// Set saves active nodes for pulse in memory.
func (a *Storage) Set(pulse pulse.Number, nodes []node.Node) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	if _, ok := a.nodes[pulse]; ok {
		return ErrOverride
	}

	if len(nodes) != 0 {
		a.nodes[pulse] = append([]node.Node{}, nodes...)
	} else {
		a.nodes[pulse] = nil
	}

	return nil
}

// All return active nodes for specified pulse.
func (a *Storage) All(pulse pulse.Number) ([]node.Node, error) {
	a.lock.RLock()
	defer a.lock.RUnlock()

	nodes, ok := a.nodes[pulse]
	if !ok {
		return nil, ErrNoNodes
	}
	res := append(nodes[:0:0], nodes...)

	return res, nil
}

// InRole return active nodes for specified pulse and role.
func (a *Storage) InRole(pulse pulse.Number, role node.StaticRole) ([]node.Node, error) {
	a.lock.RLock()
	defer a.lock.RUnlock()

	nodes, ok := a.nodes[pulse]
	if !ok {
		return nil, ErrNoNodes
	}
	var inRole []node.Node
	for _, node := range nodes {
		if node.Role == role {
			inRole = append(inRole, node)
		}
	}

	return inRole, nil
}

// DeleteForPN erases nodes for specified pulse.
func (a *Storage) DeleteForPN(pulse pulse.Number) {
	a.lock.Lock()
	defer a.lock.Unlock()

	delete(a.nodes, pulse)
}
