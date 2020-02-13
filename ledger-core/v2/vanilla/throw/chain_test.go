package throw

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type errType1 struct {
	m string
}

func (errType1) Error() string {
	return ""
}

type errType2 struct {
	m func() // incomparable
}

func (errType2) Error() string {
	return ""
}

func TestEqual(t *testing.T) {
	require.True(t, Equal(errType1{}, errType1{}))
	require.False(t, Equal(errType1{"A"}, errType1{"B"}))

	require.False(t, Equal(errType2{}, errType2{}))

	require.False(t, Equal(errType1{}, errType2{}))
	require.False(t, Equal(errType2{}, errType1{}))
}
