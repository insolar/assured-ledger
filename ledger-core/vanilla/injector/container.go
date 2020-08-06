// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package injector

import (
	"fmt"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewContainer(parentRegistry DependencyRegistry) *Container {
	return &Container{ parentRegistry: parentRegistry }
}

type Container struct {
	parentRegistry DependencyRegistry
	localRegistry sync.Map
}

func (m *Container) FindDependency(id string) (interface{}, bool) {
	if v, ok := m.localRegistry.Load(id); ok {
		return v, true
	}
	if m.parentRegistry != nil {
		return m.parentRegistry.FindDependency(id)
	}
	return nil, false
}

func AddDependency(m DependencyContainer, v interface{}) {
	if !m.TryPutDependency(GetDefaultInjectionID(v), v) {
		panic(fmt.Errorf("duplicate dependency: %T %[1]v", v))
	}
}

func AddInterfaceDependency(m DependencyContainer, v interface{}) {
	vv, vt := GetInterfaceTypeAndValue(v)
	if !m.TryPutDependency(GetDefaultInjectionIDByType(vt), vv) {
		panic(fmt.Errorf("duplicate dependency: %T %[1]v", v))
	}
}

func (m *Container) AddDependency(v interface{}) {
	AddDependency(m, v)
}

func (m *Container) AddInterfaceDependency(v interface{}) {
	AddInterfaceDependency(m, v)
}

func (m *Container) PutDependency(id string, v interface{}) {
	if id == "" {
		panic(throw.IllegalValue())
	}
	m.localRegistry.Store(id, v)
}

func (m *Container) TryPutDependency(id string, v interface{}) bool {
	if id == "" {
		panic(throw.IllegalValue())
	}
	_, loaded := m.localRegistry.LoadOrStore(id, v)
	return !loaded
}

