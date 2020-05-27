package payload

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/contract"
)

func TestVcallRequestFlags(t *testing.T)  {
	require.Equal(t, CallRequestFlags(0x0), BuildCallRequestFlags(contract.SendResultDefault, contract.CallDefault))
	require.Equal(t, CallRequestFlags(0x1), BuildCallRequestFlags(contract.SendResultFull, contract.CallDefault))
	require.Equal(t, CallRequestFlags(0x2), BuildCallRequestFlags(contract.SendResultDefault, contract.RepeatedCall))
	require.Equal(t, CallRequestFlags(0x3), BuildCallRequestFlags(contract.SendResultFull, contract.RepeatedCall))

	assert.Panics(t, func() {BuildCallRequestFlags(2, contract.CallDefault)})
	assert.Panics(t, func() {BuildCallRequestFlags(contract.SendResultDefault, 2)})
	assert.Panics(t, func() {BuildCallRequestFlags(2, 2)})

	flags := BuildCallRequestFlags(contract.SendResultDefault, contract.CallDefault)
	flags.WithRepeatedCall(contract.RepeatedCall)
	flags.WithRepeatedCall(contract.SendResultFullFlagCount)
	flags.Equal(CallRequestFlags(0x3))
	flags.Equal(BuildCallRequestFlags(contract.SendResultFullFlagCount, contract.RepeatedCall))

}
