package datawriter

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type UnresolvedDependencyMap = map[reference.Global]lineage.UnresolvedDependency

type DependencyTracker struct {
	pendings UnresolvedDependencyMap
	links    map[reference.Global]*dependencyLink
}

type dependencyLink struct {
	next         *dependencyLink
	callback     *callbackInfo
	expectedRoot reference.Holder
}

func (p *DependencyTracker) GetPendingUnresolved() UnresolvedDependencyMap {
	if p.pendings == nil {
		r := p.pendings
		p.pendings = nil
		return r
	}
	return nil
}

func (p *DependencyTracker) AddDependencySet(deps []lineage.UnresolvedDependency, callbackFn func()) {
	switch {
	case callbackFn == nil:
		panic(throw.IllegalValue())
	case len(deps) == 0:
		panic(throw.IllegalValue())
	}
	cbi := &callbackInfo{}

	for _, dep := range deps {
		kh := dep.Key()
		if reference.IsEmpty(kh) {
			panic(throw.IllegalValue())
		}
		k := reference.Copy(kh)

		if p.pendings == nil {
			p.pendings = map[reference.Global]lineage.UnresolvedDependency{}
		}

		if _, ok := p.pendings[k]; !ok {
			p.pendings[k] = dep
		}

		if p.links == nil {
			p.links = map[reference.Global]*dependencyLink{}
		}

		cbi.dependencyCount++
		link := &dependencyLink{ callback: cbi, expectedRoot: dep.RecapRootRef}
		link.next = p.links[k]
		p.links[k] = link
	}

	cbi.callbackFn = callbackFn
}

func (p *DependencyTracker) ResolveDependency(ref reference.Global) {
	link := p.links[ref]

	delete(p.links, ref)
	if p.links != nil && len(p.links) == 0 {
		p.links = nil
	}

	delete(p.pendings, ref)
	if p.pendings != nil && len(p.pendings) == 0 {
		p.pendings = nil
	}

	for ;link != nil; link = link.next {
		// link.expectedRoot
		link.callback.trigger()
	}
}

type callbackInfo struct {
	dependencyCount int
	callbackFn func()
}

func (p *callbackInfo) trigger() {
	p.dependencyCount--
	if p.dependencyCount == 0 && p.callbackFn != nil {
		p.callbackFn()
	}
}

