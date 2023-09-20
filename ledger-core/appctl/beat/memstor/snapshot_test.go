package memstor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSnapshot_GetPulse(t *testing.T) {
	snapshot := NewSnapshot(10, nil)
	assert.EqualValues(t, 10, snapshot.GetPulseNumber())
	snapshot = NewSnapshot(152, nil)
	assert.EqualValues(t, 152, snapshot.GetPulseNumber())
}
