package reference

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalMarshallUnmarshall(t *testing.T) {

	l1 := fixedSelfReference().GetLocal()
	buff, err := MarshalLocalJSON(l1)
	assert.NoError(t, err)

	l2, err := UnmarshalLocalJSON(buff)
	assert.NoError(t, err)
	assert.Equal(t, 0, l1.Compare(l2.GetLocal()))

	l1Buf, err := MarshalLocal(l1)
	assert.NoError(t, err)

	buff2 := make([]byte, 32)
	count, err := MarshalLocalTo(l1, buff2)
	assert.NoError(t, err)
	assert.Equal(t, count, len(l1Buf))

	l2, err = UnmarshalLocal(l1Buf)
	assert.NoError(t, err)

	assert.Equal(t, 0, l1.Compare(l2.GetLocal()))
}

func TestMarshallUnmarshall(t *testing.T) {

	g1 := fixedSelfReference()
	buff, err := MarshalJSON(g1)
	assert.NoError(t, err)

	g2, err := UnmarshalJSON(buff)
	assert.NoError(t, err)
	assert.Equal(t, 0, g1.Compare(g2))

	l1Buf, err := Marshal(g1)
	assert.NoError(t, err)

	buff2 := make([]byte, 64)
	count, err := MarshalTo(g1, buff2)
	assert.NoError(t, err)
	assert.Equal(t, count, len(l1Buf))

	b, l, err := Unmarshal(l1Buf)
	assert.NoError(t, err)

	assert.Equal(t, 0, g1.Compare(New(b, l)))
}
