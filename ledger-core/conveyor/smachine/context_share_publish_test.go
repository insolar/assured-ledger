// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGlobalPublishWithoutRegistry(t *testing.T) {
	sm := SlotMachine{}
	slot := Slot{ idAndStep: numberOfReservedSteps*stepIncrement|1}
	slot.machine = &sm
	ctx := slotContext{}
	ctx.s = &slot
	ctx.mode = updCtxExec

	key := "test"
	require.Nil(t, ctx.GetPublished(key))
	link := ctx.GetPublishedGlobalAlias(key)
	require.True(t, link.IsZero())

	require.True(t, ctx.PublishGlobalAlias(key))

	require.Nil(t, ctx.GetPublished(key))
	link = ctx.GetPublishedGlobalAlias(key)
	require.False(t, link.IsZero())
	require.Equal(t, slot.NewLink(), link)
}
