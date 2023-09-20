package ctlsection

import (
	"encoding/binary"
	"sort"

	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type ordinalList []ledger.Ordinal
const ordinalSize = 4

func (v ordinalList) Size() int {
	return len(v)<<2
}

func (v ordinalList) Write(b []byte) (n int, err error) {
	for _, o := range v {
		binary.LittleEndian.PutUint32(b[n:], uint32(o))
		n += ordinalSize
	}
	return n, err
}

func SortOrdinals(list []ledger.Ordinal) {
	sort.Sort(ordinalSorter(list))
}

type ordinalSorter []ledger.Ordinal

func (v ordinalSorter) Len() int {
	return len(v)
}

func (v ordinalSorter) Less(i, j int) bool {
	return v[i] < v[j]
}

func (v ordinalSorter) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func NewOrdinalListMapper(b []byte) OrdinalListMapper {
	m := OrdinalListMapper(b)
	if m.Len() < 0 {
		panic(throw.IllegalValue())
	}
	return m
}

type OrdinalListMapper []byte

func (v OrdinalListMapper) Len() int {
	switch n := len(v); {
	case n == 0:
		return 0
	case n & (ordinalSize - 1) == 0:
		return n >> 2
	default:
		return -1
	}
}

func (v OrdinalListMapper) TryGet(ord ledger.Ordinal) (ledger.Ordinal, bool) {
	if int(ord<<2) >= len(v) {
		return 0, false
	}
	return ledger.Ordinal(binary.LittleEndian.Uint32(v[ord<<2:])), true
}

