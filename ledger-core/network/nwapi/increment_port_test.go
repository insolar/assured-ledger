package nwapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIncrementPort(t *testing.T) {
	addr := IncrementPort("0.0.0.0:8080")
	assert.Equal(t, "0.0.0.0:8081", addr)

	addr = IncrementPort("[::]:8080")
	assert.Equal(t, "[::]:8081", addr)

	addr = IncrementPort("0.0.0.0:0")
	assert.Equal(t, "0.0.0.0:0", addr)

	assert.Panics(t, func() {
		IncrementPort("invalid_address")
	})

	addr = IncrementPort("127.0.0.1:port")
	assert.Equal(t, "127.0.0.1:port", addr)
}
