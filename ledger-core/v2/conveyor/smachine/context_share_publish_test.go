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
	require.True(t, ctx.PublishGlobalAlias("test"))
	link := ctx.GetPublishedGlobalAlias("test")
	require.Equal(t, slot.NewLink(), link)
}
