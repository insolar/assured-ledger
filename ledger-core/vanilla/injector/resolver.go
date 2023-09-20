package injector

import (
	"reflect"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewDependencyResolver(target interface{}, globalParent DependencyRegistry, localParent DependencyRegistry,
	onPutFn func(id string, v interface{}, from DependencyOrigin)) DependencyResolver {
	if target == nil {
		panic(throw.IllegalValue())
	}
	return DependencyResolver{target: target, globalParent: globalParent, localParent: localParent, onPutFn: onPutFn}
}

type DependencyOrigin uint8

const (
	DependencyFromLocal DependencyOrigin = 1 << iota
	DependencyFromProvider
)

type DependencyResolver struct {
	globalParent DependencyRegistry
	localParent  DependencyRegistry
	target       interface{}
	implMap      map[reflect.Type]interface{}
	resolved     map[string]interface{}
	onPutFn      func(id string, v interface{}, from DependencyOrigin)
}

func (p *DependencyResolver) IsZero() bool {
	return p.target == nil
}

func (p *DependencyResolver) IsEmpty() bool {
	return len(p.resolved) == 0
}

func (p *DependencyResolver) Count() int {
	return len(p.resolved)
}

func (p *DependencyResolver) Target() interface{} {
	return p.target
}

func (p *DependencyResolver) ResolveAndReplace(overrides map[string]interface{}) {
	for id, v := range overrides {
		if id == "" {
			panic(throw.IllegalValue())
		}
		p.resolveAndPut(id, v, DependencyFromLocal)
	}
}

func (p *DependencyResolver) ResolveAndMerge(values map[string]interface{}) {
	for id, v := range values {
		if id == "" {
			panic(throw.IllegalValue())
		}
		if _, ok := p.resolved[id]; ok {
			continue
		}
		p.resolveAndPut(id, v, DependencyFromLocal)
	}
}

func (p *DependencyResolver) FindDependency(id string) (interface{}, bool) {
	if id == "" {
		panic(throw.IllegalValue())
	}
	if v, ok := p.resolved[id]; ok { // allows nil values
		return v, true
	}

	v, ok, _ := p.getFromParent(id)
	return v, ok
}

func (p *DependencyResolver) FindImplementation(t reflect.Type, checkAmbiguous bool) (interface{}, error) {
	if v := p.implMap[t]; v != nil {
		return v, nil
	} else if p.implMap == nil {
		p.implMap = map[reflect.Type]interface{}{}
	}

	var from DependencyOrigin
	id := ""
	var v interface{}

	if scanner, ok := p.localParent.(ScanDependencyRegistry); ok {
		var err error
		id, v, err = p.findImpl(scanner, t, checkAmbiguous, "")
		switch {
		case err != nil:
			return nil, err
		case v != nil:
			from |= DependencyFromLocal
		}
	}

	switch scanner, ok := p.globalParent.(ScanDependencyRegistry); {
	case !ok:
	case v == nil:
		var err error
		id, v, err = p.findImpl(scanner, t, checkAmbiguous, "")
		switch {
		case err != nil:
			return nil, err
		case v == nil:
			return nil, nil
		}
	case checkAmbiguous:
		if _, _, err := p.findImpl(scanner, t, true, id); err != nil {
			return nil, err
		}
	}

	if from&DependencyFromLocal != 0 {
		p.putResolved(id, v, from)
	}
	p.implMap[t] = v

	return v, nil
}

func (p *DependencyResolver) findImpl(scanner ScanDependencyRegistry, t reflect.Type, checkAmbiguous bool, fid string) (id string, v interface{}, err error) {
	id = fid
	scanner.ScanDependencies(func(xid string, xv interface{}) bool {
		switch {
		case xv == nil || xid == "":
			return false
		case !reflect.ValueOf(xv).Type().AssignableTo(t):
			return false
		case id != "":
			err = throw.E("ambiguous dependency", struct {
				ExpectedType reflect.Type
				ID1, ID2 string
			}{
				t, id, xid,
			})
			return true
		default:
			id, v = xid, xv
			return !checkAmbiguous
		}
	})
	return
}

func (p *DependencyResolver) GetResolvedDependency(id string) (interface{}, bool) {
	if id == "" {
		panic(throw.IllegalValue())
	}
	return p.getResolved(id)
}

func (p *DependencyResolver) getFromParent(id string) (interface{}, bool, DependencyOrigin) {
	if p.localParent != nil {
		if v, ok := p.localParent.FindDependency(id); ok {
			return v, true, DependencyFromLocal
		}
	}
	if p.globalParent != nil {
		if v, ok := p.globalParent.FindDependency(id); ok {
			return v, true, 0
		}
	}
	return nil, false, 0
}

func (p *DependencyResolver) getResolved(id string) (interface{}, bool) {
	if v, ok := p.resolved[id]; ok { // allows nil values
		return v, true
	}
	if v, ok, from := p.getFromParent(id); ok {
		return p.resolveAndPut(id, v, from), true
	}

	return nil, false
}

func (p *DependencyResolver) resolveAndPut(id string, v interface{}, from DependencyOrigin) interface{} {
	if dp, ok := v.(DependencyProviderFunc); ok {
		p._putResolved(id, nil) // guard for resolve loop
		v = dp(p.target, id, p.GetResolvedDependency)
		p.putResolved(id, v, from|DependencyFromProvider)
	} else if from&DependencyFromLocal != 0 {
		p.putResolved(id, v, from)
	}
	return v
}

func (p *DependencyResolver) putResolved(id string, v interface{}, from DependencyOrigin) {
	p._putResolved(id, v)
	if p.onPutFn != nil {
		p.onPutFn(id, v, from)
	}
}

func (p *DependencyResolver) _putResolved(id string, v interface{}) {
	if p.resolved == nil {
		p.resolved = make(map[string]interface{})
	}
	p.resolved[id] = v
}

func (p *DependencyResolver) CopyResolved() map[string]interface{} {
	n := len(p.resolved)
	if n == 0 {
		return nil
	}
	result := make(map[string]interface{}, n)

	for id, v := range p.resolved {
		result[id] = v
	}

	return result
}

func (p *DependencyResolver) Flush() map[string]interface{} {
	n := len(p.resolved)
	if n == 0 {
		return nil
	}
	m := p.resolved
	p.resolved = nil
	return m
}
