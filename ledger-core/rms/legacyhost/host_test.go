package legacyhost

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
)

func TestNewHost(t *testing.T) {
	actualHost, _ := NewHost("127.0.0.1:31337")
	expectedHost, _ := NewHost("127.0.0.1:31337")

	require.Equal(t, expectedHost, actualHost)
}

func TestNewHost_Error(t *testing.T) {
	_, err := NewHost("invalid_addr")

	require.Error(t, err)
}

func TestNewHostN(t *testing.T) {
	ref := gen.UniqueGlobalRef()

	actualHost, _ := NewHostN("127.0.0.1:31337", ref)
	expectedHost, _ := NewHostN("127.0.0.1:31337", ref)

	require.True(t, actualHost.NodeID.Equal(ref))
	require.True(t, expectedHost.NodeID.Equal(ref))

	require.Equal(t, expectedHost, actualHost)
}

func TestNewHostN_Error(t *testing.T) {
	_, err := NewHostN("invalid_addr", gen.UniqueGlobalRef())

	require.Error(t, err)
}

func TestNewHostNS(t *testing.T) {
	ref := gen.UniqueGlobalRef()
	shortID := node.ShortNodeID(123)

	actualHost, _ := NewHostNS("127.0.0.1:31337", ref, shortID)
	expectedHost, _ := NewHostNS("127.0.0.1:31337", ref, shortID)

	require.Equal(t, actualHost.ShortID, shortID)
	require.Equal(t, expectedHost.ShortID, shortID)

	require.Equal(t, expectedHost, actualHost)
}

func TestNewHostNS_Error(t *testing.T) {
	_, err := NewHostNS("invalid_addr", gen.UniqueGlobalRef(), node.ShortNodeID(123))

	require.Error(t, err)
}

func TestHost_String(t *testing.T) {
	nd, _ := NewHost("127.0.0.1:31337")
	nd.NodeID = gen.UniqueGlobalRef()
	string := "id: " + fmt.Sprintf("%d", nd.ShortID) + " ref: " + nd.NodeID.String() + " addr: " + nd.Address.String()

	require.Equal(t, string, nd.String())
}

func TestHost_Equal(t *testing.T) {
	id1 := gen.UniqueGlobalRef()
	id2 := gen.UniqueGlobalRef()
	idNil := reference.Global{}
	addr1, _ := NewAddress("127.0.0.1:31337")
	addr2, _ := NewAddress("10.10.11.11:12345")

	require.NotEqual(t, id1, id2)

	tests := []struct {
		id1   reference.Global
		addr1 Address
		id2   reference.Global
		addr2 Address
		equal bool
		name  string
	}{
		{id1, addr1, id1, addr1, true, "same id and address"},
		{id1, addr1, id1, addr2, false, "different addresses"},
		{id1, addr1, id2, addr1, false, "different ids"},
		{id1, addr1, id2, addr2, false, "different id and address"},
		{id1, addr1, id2, addr2, false, "different id and address"},
		{id1, Address{}, id1, Address{}, true, "empty addresses"},
		{idNil, addr1, idNil, addr1, true, "nil ids"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h1 := Host{NodeID: test.id1, Address: test.addr1}
			h2 := Host{NodeID: test.id2, Address: test.addr2}
			if test.equal != h1.Equal(h2) {
				if test.equal {
					require.Equal(t, h1, h2)
				} else {
					require.NotEqual(t, h1, h2)
				}
			}
		})
	}
}

func marshalUnmarshalHost(t *testing.T, h *Host) *Host {
	data, err := h.Marshal()
	require.NoError(t, err)
	h2 := Host{}
	err = h2.Unmarshal(data)
	require.NoError(t, err)
	return &h2
}

func TestHost_Marshal(t *testing.T) {
	ref := gen.UniqueGlobalRef()
	sid := node.ShortNodeID(137)
	h := Host{}
	h.NodeID = ref
	h.ShortID = sid

	h2 := marshalUnmarshalHost(t, &h)

	assert.Equal(t, h.NodeID, h2.NodeID)
	assert.Equal(t, h.ShortID, h2.ShortID)
	assert.True(t, h.Address.IsZero())
}

func TestHost_Marshal2(t *testing.T) {
	ref := gen.UniqueGlobalRef()
	sid := node.ShortNodeID(138)
	ip := []byte{10, 11, 0, 56}
	port := 5432
	zone := "what is it for?"
	addr := Address{UDPAddr: net.UDPAddr{IP: ip, Port: port, Zone: zone}}
	h := Host{NodeID: ref, ShortID: sid, Address: addr}

	h2 := marshalUnmarshalHost(t, &h)

	assert.Equal(t, h.NodeID, h2.NodeID)
	assert.Equal(t, h.ShortID, h2.ShortID)
	require.NotNil(t, h2.Address)
	assert.Equal(t, h.Address.IP, h2.Address.IP)
	assert.Equal(t, h.Address.Port, h2.Address.Port)
	assert.Equal(t, h.Address.Zone, h2.Address.Zone)
}
