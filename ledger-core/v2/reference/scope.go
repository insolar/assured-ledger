// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reference

type SubScope uint8

const (
	SubScopeLifeline SubScope = baseScopeLifeline
	SubScopeLocal    SubScope = baseScopeLocalDomain
	SubScopeGlobal   SubScope = baseScopeGlobal
)

func (v SubScope) AsBaseOf(o SubScope) Scope {
	return Scope(v<<2 | o)
}

type Scope uint8

const ( // super-scopes
	baseScopeLifeline = iota
	baseScopeLocalDomain
	baseScopeReserved
	baseScopeGlobal
)

const ( // super-scopes
	superScopeLifeline    Scope = 0x04 * baseScopeLifeline
	superScopeLocalDomain Scope = 0x04 * baseScopeLocalDomain
	superScopeGlobal      Scope = 0x04 * baseScopeGlobal
)

const SuperScopeMask = 0x0C
const SubScopeMask = 0x03

const (
	LifelineRecordOrSelf Scope = superScopeLifeline + iota
	LifelinePrivateChild
	LifelinePublicChild
	LifelineDelegate
)

const (
	LocalDomainMember Scope = superScopeLocalDomain + iota
	LocalDomainPrivatePolicy
	LocalDomainPublicPolicy
	_
)

const (
	_ Scope = superScopeGlobal + iota
	_
	GlobalDomainPublicPolicy
	GlobalDomainMember
)

func (v Scope) IsLocal() bool {
	return v&SuperScopeMask <= superScopeLocalDomain
}

func (v Scope) IsOfLifeline() bool {
	return v&SuperScopeMask == superScopeLifeline
}

func (v Scope) IsOfLocalDomain() bool {
	return v&SuperScopeMask == superScopeLocalDomain
}

func (v Scope) IsGlobal() bool {
	return v&SuperScopeMask == superScopeGlobal
}

func (v Scope) SuperScope() SubScope {
	return SubScope(v >> 2)
}

func (v Scope) SubScope() SubScope {
	return SubScope(v & SubScopeMask)
}

func (v Scope) Scope() Scope {
	return v
}

type SelfScope uint8

const (
	SelfScopeObject       SelfScope = baseScopeLifeline
	SelfScopeLocalDomain  SelfScope = baseScopeLocalDomain
	SelfScopeGlobalDomain SelfScope = baseScopeGlobal
)

func (v SelfScope) SuperScope() SubScope {
	return SubScope(v)
}

func (v SelfScope) SubScope() SubScope {
	return SubScope(v)
}

func (v SelfScope) Scope() Scope {
	return Scope(v | v<<2)
}
