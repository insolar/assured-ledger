package reference

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelfRefTemplate(t *testing.T) {
	tm := NewSelfRefTemplate(100_000, SelfScopeLifeline)
	require.False(t, tm.IsZero())
	require.Equal(t, LifelineRecordOrSelf, tm.GetScope())
	require.True(t, tm.HasBase())
	require.True(t, tm.CanAsRecord())

	h := CopyToLocalHash(hash256())
	require.Equal(t, NewSelf(NewLocal(100_000, SubScopeLifeline, h)), tm.WithHash(h))
	require.Equal(t, NewLocal(100_000, SubScopeLifeline, h), tm.WithHashAsRecord(h))
}

func TestRecordRefTemplate(t *testing.T) {
	tm := NewRecordRefTemplate(100_000)
	require.False(t, tm.IsZero())
	require.Equal(t, LifelineRecordOrSelf, tm.GetScope())
	require.False(t, tm.HasBase())
	require.True(t, tm.CanAsRecord())

	h := CopyToLocalHash(hash256())
	require.Equal(t, NewLocal(100_000, SubScopeLifeline, h), tm.WithHashAsRecord(h))
	require.Panics(t, func() { tm.WithHash(h) })
}

func TestRefTemplate(t *testing.T) {
	h := CopyToLocalHash(hash256())
	g := New(NewLocal(100_000, SubScopeLifeline, h), NewLocal(100_001, SubScopeGlobal, h))

	tm := NewRefTemplate(g, 100_002)
	require.False(t, tm.IsZero())
	require.Equal(t, LifelineDelegate, tm.GetScope())
	require.True(t, tm.HasBase())
	require.False(t, tm.CanAsRecord())

	h2 := h
	h2[1] = 99
	require.Equal(t,
		New(
			NewLocal(100_000, SubScopeLifeline, h),
			NewLocal(100_002, SubScopeGlobal, h2)),
		tm.WithHash(h2))

	require.Panics(t, func() { tm.WithHashAsRecord(h) })
}

func TestMutableSelfRefTemplate(t *testing.T) {
	tt := NewSelfRefTemplate(100_000, SelfScopeLifeline)
	tm := tt.AsMutable()

	require.False(t, tm.IsZero())
	require.False(t, tm.HasHash())
	require.Equal(t, LifelineRecordOrSelf, tm.GetScope())
	require.True(t, tm.HasBase())
	require.True(t, tm.CanAsRecord())

	h := CopyToLocalHash(hash256())
	require.Equal(t, NewSelf(NewLocal(100_000, SubScopeLifeline, h)), tm.WithHash(h))
	require.Equal(t, NewLocal(100_000, SubScopeLifeline, h), tm.WithHashAsRecord(h))

	require.Panics(t, func() { tm.GetBase() })
	require.Panics(t, func() { tm.GetLocal() })

	h2 := h
	h2[1] = 99
	tm.SetHash(h2)
	require.True(t, tm.HasHash())

	require.Equal(t, NewSelf(NewLocal(100_000, SubScopeLifeline, h2)), tm.MustGlobal())
	require.Equal(t, NewLocal(100_000, SubScopeLifeline, h2), tm.MustRecord())
	require.True(t, Equal(NewSelf(NewLocal(100_000, SubScopeLifeline, h2)), &tm))

	require.Equal(t, NewSelf(NewLocal(100_000, SubScopeLifeline, h)), tm.WithHash(h))
	require.Equal(t, NewLocal(100_000, SubScopeLifeline, h), tm.WithHashAsRecord(h))
	require.Equal(t, tt, tm.AsTemplate())
}

func TestMutableRecordRefTemplate(t *testing.T) {
	tt := NewRecordRefTemplate(100_000)
	tm := tt.AsMutable()

	require.False(t, tm.IsZero())
	require.Equal(t, LifelineRecordOrSelf, tm.GetScope())
	require.False(t, tm.HasBase())
	require.True(t, tm.CanAsRecord())

	h := CopyToLocalHash(hash256())
	require.Equal(t, NewLocal(100_000, SubScopeLifeline, h), tm.WithHashAsRecord(h))

	require.Panics(t, func() { tm.WithHash(h) })
	require.Panics(t, func() { tm.MustGlobal() })
	require.Panics(t, func() { tm.MustRecord() })

	require.Panics(t, func() { tm.GetBase() })
	require.Panics(t, func() { tm.GetLocal() })

	h2 := h
	h2[1] = 99
	tm.SetHash(h2)
	require.True(t, tm.HasHash())

	require.Panics(t, func() { tm.MustGlobal() })
	require.Equal(t, NewLocal(100_000, SubScopeLifeline, h2), tm.MustRecord())
	require.True(t, Equal(New(Local{}, NewLocal(100_000, SubScopeLifeline, h2)), &tm))

	require.Panics(t, func() { tm.WithHash(h) })
	require.Equal(t, NewLocal(100_000, SubScopeLifeline, h), tm.WithHashAsRecord(h))
	require.Equal(t, tt, tm.AsTemplate())
}

func TestMutableRefTemplate(t *testing.T) {
	h := CopyToLocalHash(hash256())
	g := New(NewLocal(100_000, SubScopeLifeline, h), NewLocal(100_001, SubScopeGlobal, h))

	tt := NewRefTemplate(g, 100_002)
	tm := tt.AsMutable()

	require.False(t, tm.IsZero())
	require.Equal(t, LifelineDelegate, tm.GetScope())
	require.True(t, tm.HasBase())
	require.False(t, tm.CanAsRecord())

	h3 := h
	h3[1] = 199

	require.Equal(t, New(g.GetBase(), NewLocal(100_002, SubScopeGlobal, h3)), tm.WithHash(h3))

	require.Panics(t, func() { tm.WithHashAsRecord(h3) })
	require.Panics(t, func() { tm.MustGlobal() })
	require.Panics(t, func() { tm.MustRecord() })

	require.Panics(t, func() { tm.GetBase() })
	require.Panics(t, func() { tm.GetLocal() })

	h2 := h
	h2[1] = 99

	tm.SetHash(h2)
	require.True(t, tm.HasHash())

	require.Equal(t, New(g.GetBase(), NewLocal(100_002, SubScopeGlobal, h2)), tm.MustGlobal())
	require.Panics(t, func() { tm.MustRecord() })
	require.Panics(t, func() { tm.WithHashAsRecord(h3) })
	require.True(t, Equal(New(g.GetBase(), NewLocal(100_002, SubScopeGlobal, h2)), &tm))

	require.Equal(t, New(g.GetBase(), NewLocal(100_002, SubScopeGlobal, h3)), tm.WithHash(h3))
	require.Equal(t, tt, tm.AsTemplate())
}

func TestMutableZeroTemplate(t *testing.T) {
	h := CopyToLocalHash(hash256())
	g := New(NewLocal(100_000, SubScopeLifeline, h), NewLocal(100_001, SubScopeGlobal, h))

	tt := NewRefTemplate(g, 100_002)
	tm := tt.AsMutable()

	tm.SetZeroValue()
	require.True(t, tm.HasHash())
	require.True(t, tm.IsZero())
	require.Equal(t, Global{}, Copy(&tm))
	require.True(t, Equal(Global{}, &tm))
	require.True(t, Equal(Global{}, tm.MustGlobal()))
	require.Equal(t, Local{}, tm.MustRecord())

	require.Panics(t, func() { tm.AsTemplate() })
}
