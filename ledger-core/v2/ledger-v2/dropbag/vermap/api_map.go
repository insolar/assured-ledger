// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package vermap

import "github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"

type Key = longbits.ByteString

type Value = []byte

type ReadMap interface {
	Get(Key) (Value, error)
	Contains(Key) bool
}

type UpdateMap interface {
	ReadMap
	Set(Key, Value) error
	SetEntry(Entry) error
}

type LiveMap interface {
	//UpdateMap
	ViewNow() ReadMap
	StartUpdate() TxMap
}

type TxMap interface {
	UpdateMap

	//GetUpdated(Key) (Value, error)
	Discard()
	Commit() error
}
