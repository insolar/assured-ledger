package nwaddr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIdentityIs(t *testing.T) {
	require.True(t, IP.IsIP())
	require.True(t, DNS.IsDNS())
	require.True(t, HostPK.IsHostPK())
	require.True(t, HostID.IsHostID())
	require.True(t, LocalUID.IsLocalUID())

	require.False(t, LocalUID.IsIP())
	require.False(t, IP.IsDNS())
	require.False(t, IP.IsHostPK())
	require.False(t, IP.IsHostID())
	require.False(t, IP.IsLocalUID())

	require.False(t, DNS.IsResolved())
	require.True(t, IP.IsResolved())
	require.True(t, IP.IsNetCompatible())
	require.True(t, DNS.IsNetCompatible())
	require.False(t, LocalUID.IsNetCompatible())
}
