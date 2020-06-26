// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc"
)

type stageNo uint32
type recordNo uint32
type filamentNo uint32

type updateStage struct {
	seqNo stageNo
	next  *updateStage // latter one

	filaments    []filamentEndings
	filamentCopy bool

	future    *buildersvc.Future
}

type updateRecord struct {
	prev recordNo
	next recordNo
	recordNo recordNo

	stageNo stageNo

	Record
}

type filamentEndings struct {
	earliest, latest recordNo
}
