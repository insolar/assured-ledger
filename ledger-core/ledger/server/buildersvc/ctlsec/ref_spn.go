// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package ctlsec

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

const SPNControlSection = pulse.Number(127)
// const SPNControlSection = pulse.Number(127)

func CtlSectionRef(id ledger.SectionID, summary bool) reference.Global {
	return reference.New(
		reference.NewLocal(SPNControlSection, 0, reference.LocalHash{}),
		reference.NewLocal(SPNControlSection, 0, reference.LocalHash{}), // TODO key
	)
}

func JetRef(id jet.ID) reference.Global {
	return reference.New(
		reference.NewLocal(SPNControlSection, 0, reference.LocalHash{}),
		reference.NewLocal(SPNControlSection, 0, reference.LocalHash{}), // TODO key
	)
}
