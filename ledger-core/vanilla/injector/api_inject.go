// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package injector

type DependencyRegistry interface {
	FindDependency(id string) (interface{}, bool)
}

type LocalDependencyRegistry interface {
	DependencyRegistry
	FindLocalDependency(id string) (interface{}, bool)
}

type DependencyRegistryFunc func(id string) (interface{}, bool)

type DependencyContainer interface {
	DependencyRegistry
	PutDependency(id string, v interface{})
	TryPutDependency(id string, v interface{}) bool
}

type DependencyProviderFunc func(target interface{}, id string, resolveFn DependencyRegistryFunc) interface{}
