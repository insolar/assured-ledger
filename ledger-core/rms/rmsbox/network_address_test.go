package rmsbox

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

func TestNetworkAddress_MarshalUnmarshal(t *testing.T) {
	h := nwapi.NewHostPort("127.0.0.1:123", false)

	na := NewNetworkAddress(h)

	data := make([]byte, na.ProtoSize())
	size, err := h.MarshalTo(data)
	require.NoError(t, err)

	na2 := NewNetworkAddress(nwapi.Address{})
	err = na2.Unmarshal(data)
	require.NoError(t, err)

	assert.Equal(t, na.Get(), na2.Get())
	assert.Equal(t, size, h.ProtoSize())
}
