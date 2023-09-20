package shareddata

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/reflectkit"
)

func TestCallSharedDataAccessor(t *testing.T) {
	require.Equal(t, unsafe.Sizeof(sharedDataAccessor{}), unsafe.Sizeof(smachine.SharedDataAccessor{}))

	data := interface{}("abc")
	accessFn := func(interface{}) bool {
		return false
	}

	sda := smachine.NewUnboundSharedData(data).PrepareAccess(accessFn)
	unwrapped := unwrap(&sda)

	require.Equal(t, data, unwrapped.link.data)
	require.Equal(t, reflectkit.CodeOf(accessFn), reflectkit.CodeOf(unwrapped.accessFn))
	require.Equal(t, smachine.ShareDataUnbound, unwrapped.link.flags)
}
