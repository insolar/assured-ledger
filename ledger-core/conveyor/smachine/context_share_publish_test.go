package smachine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func stubExecContext() slotContext {
	sm := SlotMachine{}
	slot := Slot{ idAndStep: numberOfReservedSteps*stepIncrement|1}
	slot.machine = &sm
	ctx := slotContext{}
	ctx.s = &slot
	ctx.mode = updCtxExec
	return ctx
}

func TestGlobalPublishWithoutRegistry(t *testing.T) {
	ctx := stubExecContext()

	key := "test"
	require.Nil(t, ctx.GetPublished(key))
	link := ctx.GetPublishedGlobalAlias(key)
	require.True(t, link.IsZero())

	require.True(t, ctx.PublishGlobalAlias(key))

	require.Nil(t, ctx.GetPublished(key))
	link = ctx.GetPublishedGlobalAlias(key)
	require.False(t, link.IsZero())
	require.Equal(t, ctx.s.NewLink(), link)
}

func TestBoundPublish(t *testing.T) {
	ctx := stubExecContext()

	key := "test"
	data := "testData"
	link := ctx.Share(data, 0)
	require.False(t, link.IsZero())
	require.True(t, link.IsValid())
	require.False(t, link.IsUnbound())

	link.PrepareAccess(func(v interface{}) (wakeup bool) {
		require.Equal(t, data, v)
		return false
	}).TryUse(&executionContext{slotContext: ctx})

	require.Nil(t, ctx.GetPublished(key))
	require.True(t, ctx.Publish(key, link))
	require.False(t, ctx.Publish(key, link))

	require.Equal(t, link, ctx.GetPublished(key))
	require.Equal(t, link, ctx.GetPublishedLink(key))
	ctx.UnpublishAll()

	require.Nil(t, ctx.GetPublished(key))
	require.True(t, ctx.Publish(key, data))
}

func TestBoundPublishUnpublish(t *testing.T) {
	ctx := stubExecContext()

	key := "test"
	data := "testData"

	require.True(t, ctx.Publish(key, data))
	require.False(t, ctx.Publish(key, data))

	require.True(t, ctx.Unpublish(key))

	require.Nil(t, ctx.GetPublished(key))
	require.True(t, ctx.Publish(key, data))
}

func TestUnboundPublish(t *testing.T) {
	ctx := stubExecContext()

	key := "test"
	data := "testData"
	link := ctx.Share(data, ShareDataUnbound)
	require.False(t, link.IsZero())
	require.True(t, link.IsValid())
	require.True(t, link.IsUnbound())

	link.PrepareAccess(func(v interface{}) (wakeup bool) {
		require.Equal(t, data, v)
		return false
	}).TryUse(&executionContext{slotContext: ctx})

	require.Nil(t, ctx.GetPublished(key))
	require.True(t, ctx.Publish(key, link))
	require.False(t, ctx.Publish(key, link))

	require.Equal(t, link, ctx.GetPublished(key))
	require.Equal(t, link, ctx.GetPublishedLink(key))
	ctx.UnpublishAll()

	require.Equal(t, link, ctx.GetPublished(key))
	require.False(t, ctx.Publish(key, data))
}
