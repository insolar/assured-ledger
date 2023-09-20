package reference

type SubScope uint8

const (
	SubScopeLifeline SubScope = baseScopeLifeline
	SubScopeLocal    SubScope = baseScopeLocalDomain
	SubScopeGlobal   SubScope = baseScopeGlobal
)

const scopeBits = 2

func (v SubScope) AsBaseOf(o SubScope) Scope {
	return Scope(v<<scopeBits | o)
}

func (v SubScope) AsSelfScope() SelfScope {
	return SelfScope(v)
}

type Scope uint8

const ( // super-scopes
	baseScopeLifeline = iota
	baseScopeLocalDomain
	baseScopeReserved
	baseScopeGlobal
)

const ( // super-scopes
	superScopeLifeline    Scope = baseScopeLifeline << scopeBits
	superScopeLocalDomain Scope = baseScopeLocalDomain << scopeBits
	superScopeGlobal      Scope = baseScopeGlobal << scopeBits
)

const SubScopeMask = 1<<scopeBits - 1
const SuperScopeMask = SubScopeMask << scopeBits

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
	SelfScopeLifeline     SelfScope = baseScopeLifeline
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
