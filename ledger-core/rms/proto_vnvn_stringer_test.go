package rms

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStringer(t *testing.T) {
	v := VCallRequest{}
	require.NotContains(t, v.String(), "PANIC")
}
