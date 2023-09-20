package reference

import (
	"encoding/base64"
	"encoding/binary"
	"sync/atomic"
	"testing"
	"time"

	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

var uniqueSeq uint32

func getUnique() uint32 {
	return atomic.AddUint32(&uniqueSeq, 1)
}

func randLocal() Local {
	var id Local

	f := fuzz.New().NilChance(0).Funcs(func(id *Local, c fuzz.Continue) {
		var hash [LocalBinaryHashSize]byte
		c.Fuzz(&hash)
		binary.BigEndian.PutUint32(hash[LocalBinaryHashSize-4:], getUnique())

		pn := pulse.OfTime(time.Now())

		*id = NewLocal(pn, 0, BytesToLocalHash(hash[:]))
	})
	f.Fuzz(&id)

	return id
}

func flipHeader(b []byte) []byte {
	// Encoder uses BigEndian, while AsBytes uses LittleEndian for the header
	// b[0], b[1], b[2], b[3] = b[3], b[2], b[1], b[0]
	return b
}

func TestFromString(t *testing.T) {
	recordID := randLocal()
	domainID := randLocal()
	refStr := "insolar:1" +
		base64.RawURLEncoding.EncodeToString(flipHeader(recordID.AsBytes())) +
		string(RecordRefIDSeparator) + "1" +
		base64.RawURLEncoding.EncodeToString(flipHeader(domainID.AsBytes()))

	expectedRef := New(domainID, recordID)
	actualRef, err := GlobalFromString(refStr)
	require.NoError(t, err)

	assert.Equal(t, expectedRef, actualRef)
}

func TestGlobalZero(t *testing.T) {
	require.True(t, Global{}.IsEmpty())
	require.True(t, Global{}.IsZero())
	require.False(t, New(Local{}, NewLocal(1, 0, LocalHash{})).IsZero())
}

func TestGlobalRecRef(t *testing.T) {
	local := randLocal().WithSubScope(0)
	ref := NewRecordRef(local)
	require.Equal(t, local, ref.AsRecordID())
	require.Equal(t, local, AsRecordID(ref))

	require.True(t, ref.IsRecordScope())
	require.False(t, ref.IsSelfScope())
	require.False(t, ref.IsGlobalScope())
	require.False(t, ref.IsLifelineScope())
	require.False(t, ref.IsLocalDomainScope())
	require.False(t, ref.IsObjectReference())
}

func TestGlobalSelfRef(t *testing.T) {
	local := randLocal().WithSubScope(0)
	ref := NewSelf(local)
	require.Equal(t, local, ref.AsRecordID())
	require.False(t, ref.IsRecordScope())
	require.True(t, ref.IsSelfScope())
	require.False(t, ref.IsGlobalScope())
	require.True(t, ref.IsLifelineScope())
	require.False(t, ref.IsLocalDomainScope())
	require.True(t, ref.IsObjectReference())
}

func TestGlobalCopy(t *testing.T) {
	require.Equal(t, Global{}, Copy(nil))

	ref := New(randLocal(), randLocal())
	require.Equal(t, ref, Copy(ref))

	rp := NewPtrHolder(ref.GetBase(), ref.GetLocal())
	require.Equal(t, ref, Copy(rp))
}
