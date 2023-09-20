package trace

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRandTraceID(t *testing.T) {
	traceID := RandID()
	require.NotEmpty(t, traceID)
}
