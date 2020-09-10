// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package ctlsec

import (
	"encoding/binary"
	"sort"

	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
)

type FilamentHead struct {
	Head  ledger.Ordinal
	bundle.FilamentInfo
}

type ordinalList []ledger.Ordinal

func (v ordinalList) Size() int {
	return len(v)<<2
}

func (v ordinalList) Write(b []byte) (n int, err error) {
	for _, o := range v {
		binary.LittleEndian.PutUint32(b[n:], uint32(o))
		n += 4
	}
	return n, err
}

type filamentHeads []FilamentHead

func (v filamentHeads) Size() int {
	return len(v)*12
}

func (v filamentHeads) Write(b []byte) (n int, err error) {
	for _, h := range v {
		binary.LittleEndian.PutUint32(b[n:], uint32(h.Head))
		n += 4
		binary.LittleEndian.PutUint32(b[n:], uint32(h.FilamentInfo.Link))
		n += 4
		binary.LittleEndian.PutUint16(b[n:], uint16(h.FilamentInfo.JetID))
		n += 2
		binary.LittleEndian.PutUint16(b[n:], uint16(h.FilamentInfo.Flags))
		n += 2
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

