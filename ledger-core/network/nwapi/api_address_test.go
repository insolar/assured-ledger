package nwapi

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddress_IsLoopback(t *testing.T) {
	require.True(t, NewHost("127.0.0.1").IsLoopback())
	require.True(t, NewHost("127.1.1.1").IsLoopback())
	require.False(t, NewHost("128.0.0.1").IsLoopback())
	require.True(t, NewHost("::1").IsLoopback())
	require.False(t, NewHost("::2").IsLoopback())
}
