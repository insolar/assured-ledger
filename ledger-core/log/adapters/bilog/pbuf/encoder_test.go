package pbuf

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrependSize(t *testing.T) {
	require.Equal(t, fieldLogEntry.TagSize(), prependFieldSize)
}
