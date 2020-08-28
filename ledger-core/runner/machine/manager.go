// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package machine

import (
	_type "github.com/insolar/assured-ledger/ledger-core/runner/machine/type"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type Manager interface {
	RegisterExecutor(_type.Type, Executor) error
	GetExecutor(_type.Type) (Executor, error)
}

type defaultManager struct {
	Executors [_type.LastID]Executor
}

func NewManager() Manager {
	return &defaultManager{}
}

// RegisterExecutor registers an executor for particular `Type`
func (m *defaultManager) RegisterExecutor(t _type.Type, e Executor) error {
	m.Executors[int(t)] = e
	return nil
}

// GetExecutor returns an executor for the `Type` if it was registered (`RegisterExecutor`),
// returns error otherwise
func (m *defaultManager) GetExecutor(t _type.Type) (Executor, error) {
	if res := m.Executors[int(t)]; res != nil {
		return res, nil
	}

	return nil, errors.Errorf("No executor registered for machine %d", int(t))
}
