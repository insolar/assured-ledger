// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package shareddata

import (
	"unsafe"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
)

type sharedDataAccessor struct {
	link struct {
		link  smachine.SlotLink
		data  interface{}
		flags smachine.ShareDataFlags
	}
	accessFn smachine.SharedDataFunc
}

func unwrap(sda *smachine.SharedDataAccessor) *sharedDataAccessor {
	return (*sharedDataAccessor)(unsafe.Pointer(sda))
}

func CallSharedDataAccessor(sda smachine.SharedDataAccessor) smachine.SharedAccessReport {
	w := unwrap(&sda)
	w.accessFn(w.link.data)
	return smachine.SharedSlotLocalAvailable
}
