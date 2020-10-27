// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package ctlsection

import (
	"encoding/binary"

	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
)

type FilamentHead struct {
	First ledger.Ordinal
	Last  ledger.Ordinal
	JetID jet.ID
	Flags ledger.DirectoryEntryFlags
}

const FilamentHeadSize = 12

type filamentHeads []FilamentHead

func (v filamentHeads) Size() int {
	return len(v)*FilamentHeadSize
}

func (v filamentHeads) Write(b []byte) (n int, err error) {
	for _, h := range v {
		binary.LittleEndian.PutUint32(b[n:], uint32(h.First))
		n += 4
		binary.LittleEndian.PutUint32(b[n:], uint32(h.Last))
		n += 4
		binary.LittleEndian.PutUint16(b[n:], uint16(h.JetID))
		n += 2
		binary.LittleEndian.PutUint16(b[n:], uint16(h.Flags))
		n += 2
	}
	return n, err
}

