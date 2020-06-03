// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package vermap

type Options uint32

const (
	CanUpdate Options = 1 << iota
	CanReplace
	CanDelete
)

const UpdateMask = CanUpdate | CanReplace | CanDelete
