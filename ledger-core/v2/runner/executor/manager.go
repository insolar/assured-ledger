// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/insolar/blob/master/LICENSE.md.

package executor

import (
	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
)

type Manager interface {
	RegisterExecutor(t insolar.MachineType, e insolar.MachineLogicExecutor) error
	GetExecutor(t insolar.MachineType) (insolar.MachineLogicExecutor, error)
}

type manager struct {
	Executors [insolar.MachineTypesLastID]insolar.MachineLogicExecutor
}

func NewManager() Manager {
	return &manager{}
}

// RegisterExecutor registers an executor for particular `MachineType`
func (m *manager) RegisterExecutor(t insolar.MachineType, e insolar.MachineLogicExecutor) error {
	m.Executors[int(t)] = e
	return nil
}

// GetExecutor returns an executor for the `MachineType` if it was registered (`RegisterExecutor`),
// returns error otherwise
func (m *manager) GetExecutor(t insolar.MachineType) (insolar.MachineLogicExecutor, error) {
	if res := m.Executors[int(t)]; res != nil {
		return res, nil
	}

	return nil, errors.Errorf("No executor registered for machine %d", int(t))
}
