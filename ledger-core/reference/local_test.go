package reference

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

func TestLocalFromString(t *testing.T) {
	id := randLocal()
	idStr := "insolar:1" + base64.RawURLEncoding.EncodeToString(id.AsBytes())
	id2, err := LocalFromString(idStr)
	require.NoError(t, err)

	assert.Equal(t, id, id2)
}

func TestLocalHeader(t *testing.T) {
	require.True(t, Local{}.IsZero())
	require.True(t, Local{}.IsEmpty())
	require.False(t, Local{}.NotEmpty())

	r0 := randLocal()
	r := r0.WithHeader(NewLocalHeader(100000, 1))

	require.False(t, r.IsZero())
	require.False(t, r.IsEmpty())
	require.True(t, r.NotEmpty())

	require.Equal(t, 100000, int(r.GetPulseNumber()))
	require.Equal(t, 1, int(r.SubScope()))
	require.Equal(t, NewLocalHeader(100000, 1), r.GetHeader())
	require.Equal(t, r0.IdentityHashBytes(), r.IdentityHashBytes())

	r = r.WithPulse(100001)
	require.Equal(t, 100001, int(r.GetPulseNumber()))
	require.Equal(t, 1, int(r.SubScope()))
	require.Equal(t, NewLocalHeader(100001, 1), r.GetHeader())
	require.Equal(t, r0.IdentityHashBytes(), r.IdentityHashBytes())

	r = r.WithSubScope(0)
	require.Equal(t, 100001, int(r.GetPulseNumber()))
	require.Equal(t, 0, int(r.SubScope()))
	require.Equal(t, NewLocalHeader(100001, 0), r.GetHeader())
	require.Equal(t, r0.IdentityHashBytes(), r.IdentityHashBytes())

	h2 := r0.IdentityHash()
	h2[10] += 99
	r = r.WithHash(h2)
	require.Equal(t, NewLocal(100001, 0, h2), r.WithHash(h2))
	require.Equal(t, h2, r.IdentityHash())
	require.Equal(t, h2[:], r.IdentityHashBytes())
}

func TestLocalShortForm(t *testing.T) {
	r := NewLocal(100000, 1, LocalHash{})
	require.Equal(t, 0, r.hashLen())
	r = NewLocal(100000, 1, LocalHash{10: 1})
	require.Equal(t, 11, r.hashLen())
}

func TestLocalWriteTo(t *testing.T) {
	buf := bytes.Buffer{}
	r := NewLocal(100000, 1, LocalHash{10: 1})
	n, err := r.WriteTo(&buf)
	require.NoError(t, err)
	require.Equal(t, int64(LocalBinarySize), n)

	s := r.AsByteString()
	require.Equal(t, uint8('\x01'), s[10+LocalBinaryPulseAndScopeSize])
	require.Equal(t, "@\x01\x86\xa0\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00", string(s))

	b := r.AsBytes()
	require.Equal(t, []byte(s), b)
	require.Equal(t, []byte(s), buf.Bytes())
}

func hash256() (v longbits.Bits256) {
	for i := range v {
		v[i] = uint8(i)
	}
	return
}

func hashLocal() (v LocalHash) {
	for i := range v {
		v[i] = uint8(i)
	}
	return
}

func TestLocalHash(t *testing.T) {
	buf := hash256()

	h := BytesToLocalHash(buf[:])
	require.Equal(t, buf[:28], h[:])
	h = BytesToLocalHash(buf[:28])
	require.Equal(t, buf[:28], h[:])

	h = CopyToLocalHash(buf)
	require.Equal(t, buf[:28], h[:])

	require.Panics(t, func() { BytesToLocalHash(make([]byte, 31)) })
	require.Panics(t, func() { BytesToLocalHash(make([]byte, 33)) })
	require.Panics(t, func() { BytesToLocalHash(make([]byte, 27)) })
	require.Panics(t, func() { BytesToLocalHash(make([]byte, 29)) })
}
