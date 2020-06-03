// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package wormmap

import "github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"

type Key = longbits.ByteString
type Value = []byte

type Entry struct {
	Key
	Value
	UserMeta uint32
}

func NewByteEntry(key, value []byte) Entry {
	return Entry{
		Key:   longbits.CopyBytes(key),
		Value: value,
	}
}

func NewEntry(key Key, value Value) Entry {
	return Entry{
		Key:   key,
		Value: value,
	}
}

func (e Entry) WithMeta(meta uint32) Entry {
	e.UserMeta = meta
	return e
}
