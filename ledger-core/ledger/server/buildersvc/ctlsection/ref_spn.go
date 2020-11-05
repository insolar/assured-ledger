// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package ctlsection

import (
	"encoding/binary"

	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

const SPNControlSection = pulse.Number(127)

func ctlRecordLocalRef(recordType uint64, id uint16) reference.Local {
	var data reference.LocalHash
	n := 8
	binary.LittleEndian.PutUint64(data[n-8:n], recordType)
	n += 2
	binary.LittleEndian.PutUint16(data[n-2:n], id)
	return reference.NewLocal(SPNControlSection, 0, data)
}

func SectionCtlRecordRef(id ledger.SectionID, recordType uint64) reference.Global {
	return reference.NewSelf(ctlRecordLocalRef(recordType, uint16(id)))
}

func JetCtlRecordRef(id jet.ID, recordType uint64) reference.Global {
	return reference.NewSelf(ctlRecordLocalRef(recordType, uint16(id)))
}
