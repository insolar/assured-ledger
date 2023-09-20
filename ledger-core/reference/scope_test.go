package reference

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScope(t *testing.T) {
	require.Equal(t, LocalDomainMember, SubScopeLocal.AsBaseOf(SubScopeLifeline))
	require.Equal(t, LocalDomainMember, LocalDomainMember.Scope())
	require.Equal(t, SubScopeLocal, LocalDomainMember.SuperScope())
	require.Equal(t, SubScopeLifeline, LocalDomainMember.SubScope())
	require.Equal(t, SelfScopeLocalDomain, SubScopeLocal.AsSelfScope())

	require.Equal(t, SubScopeLocal, SelfScopeLocalDomain.SuperScope())
	require.Equal(t, SubScopeLocal, SelfScopeLocalDomain.SubScope())
	require.Equal(t, LocalDomainPrivatePolicy, SelfScopeLocalDomain.Scope())

	require.True(t, GlobalDomainMember.IsGlobal())
	require.False(t, GlobalDomainMember.IsLocal())
	require.False(t, GlobalDomainMember.IsOfLifeline())
	require.False(t, GlobalDomainMember.IsOfLocalDomain())

	require.False(t, LocalDomainMember.IsGlobal())
	require.True(t, LocalDomainMember.IsLocal())
	require.False(t, LocalDomainMember.IsOfLifeline())
	require.True(t, LocalDomainMember.IsOfLocalDomain())

	require.False(t, LifelineRecordOrSelf.IsGlobal())
	require.True(t, LifelineRecordOrSelf.IsLocal())
	require.True(t, LifelineRecordOrSelf.IsOfLifeline())
	require.False(t, LifelineRecordOrSelf.IsOfLocalDomain())
}
