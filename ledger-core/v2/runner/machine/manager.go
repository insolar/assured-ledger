// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package machine

import (
	errors "github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type Manager interface {
	RegisterExecutor(Type, Executor) error
	GetExecutor(Type) (Executor, error)
}

type defaultManager struct {
	Executors [LastID]Executor
}

func NewManager() Manager {
	return &defaultManager{}
}

// RegisterExecutor registers an executor for particular `Type`
func (m *defaultManager) RegisterExecutor(t Type, e Executor) error {
	m.Executors[int(t)] = e
	return nil
}

// GetExecutor returns an executor for the `Type` if it was registered (`RegisterExecutor`),
// returns error otherwise
func (m *defaultManager) GetExecutor(t Type) (Executor, error) {
	if res := m.Executors[int(t)]; res != nil {
		return res, nil
	}

	return nil, errors.Errorf("No executor registered for machine %d", int(t))
}
