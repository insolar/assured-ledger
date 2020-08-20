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

func NewDynamicContainer(parentRegistry DependencyRegistry) *DynamicContainer {
	return &DynamicContainer{ parentRegistry: parentRegistry }
}

type DynamicContainer struct {
	parentRegistry DependencyRegistry
	localRegistry sync.Map
}

func (m *DynamicContainer) FindDependency(id string) (interface{}, bool) {
	if m == nil {
		return nil, false
	}

	if v, ok := m.localRegistry.Load(id); ok {
		return v, true
	}
	if m.parentRegistry != nil {
		return m.parentRegistry.FindDependency(id)
	}
	return nil, false
}

func (m *DynamicContainer) ScanDependencies(fn func(id string, v interface{}) bool) (found bool) {
	if m == nil {
		return false
	}

	if fn == nil {
		panic(throw.IllegalValue())
	}

	m.localRegistry.Range(func(key, value interface{}) bool {
		if fn(key.(string), value) {
			found = true
			return false
		}
		return true
	})

	if found {
		return true
	}

	if sp, ok := m.parentRegistry.(ScanDependencyRegistry); ok {
		return sp.ScanDependencies(fn)
	}
	return false
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

func NameOfDependency(v interface{}) string {
	return GetDefaultInjectionID(v)
}

func NameOfInterfaceDependency(v interface{}) string {
	_, vt := GetInterfaceTypeAndValue(v)
	return GetDefaultInjectionIDByType(vt)
}

func (m *DynamicContainer) AddDependency(v interface{}) {
	AddDependency(m, v)
}

func (m *DynamicContainer) AddInterfaceDependency(v interface{}) {
	AddInterfaceDependency(m, v)
}

func (m *DynamicContainer) PutDependency(id string, v interface{}) {
	if id == "" {
		panic(throw.IllegalValue())
	}
	m.localRegistry.Store(id, v)
}

func (m *DynamicContainer) TryPutDependency(id string, v interface{}) bool {
	if id == "" {
		panic(throw.IllegalValue())
	}
	_, loaded := m.localRegistry.LoadOrStore(id, v)
	return !loaded
}

func (m *DynamicContainer) DeleteDependency(id string) {
	m.localRegistry.Delete(id)
}

func (m *DynamicContainer) CopyAsStatic() StaticContainer {
	if m == nil {
		return NewStaticContainer(nil, nil)
	}
	return NewStaticContainer(m.parentRegistry, m)
}

func (m *DynamicContainer) ReplaceDependency(v interface{}) {
	id := GetDefaultInjectionID(v)
	m.localRegistry.Delete(id)
	m.PutDependency(id, v)
}

func (m *DynamicContainer) ReplaceInterfaceDependency(v interface{}) {
	vv, vt := GetInterfaceTypeAndValue(v)
	id := GetDefaultInjectionIDByType(vt)
	m.localRegistry.Delete(id)
	m.PutDependency(id, vv)
}

