// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l2

import "github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"

type Address struct {
	flags addressFlags
	port  uint16
	addr  longbits.ByteString
}

type addressFlags uint16

const (
	addressName addressFlags = 1 << iota
)
