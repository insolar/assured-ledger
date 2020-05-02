// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExample(t *testing.T) {
	m := MessageExample{MsgParam: 11}
	require.Equal(t, 99, int(m.GetDefaultPolymorphID()))
	require.Equal(t, 199, int(m.RecordExample.GetDefaultPolymorphID()))

	require.Equal(t, 11, int(m.GetMsgParam()))
	h := m.AsHead()
	require.Equal(t, 11, int(h.GetMsgParam()))
	require.Equal(t, 99, int(h.(*MessageExample_Head).GetDefaultPolymorphID()))
}

func (m *MessageExample) AsHead() MessageExampleHead {
	return (*MessageExample_Head)(m)
}
