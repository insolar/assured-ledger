package injector

type DependencyRegistry interface {
	FindDependency(id string) (interface{}, bool)
}

type ScanDependencyRegistry interface {
	ScanDependencies(fn func(id string, v interface{}) bool) bool
}

type DependencyContainer interface {
	DependencyRegistry
	PutDependency(id string, v interface{})
	TryPutDependency(id string, v interface{}) bool
}

type DependencyRegistryFunc func(id string) (interface{}, bool)
type DependencyProviderFunc func(target interface{}, id string, resolveFn DependencyRegistryFunc) interface{}
