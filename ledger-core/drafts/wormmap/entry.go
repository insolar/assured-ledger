package wormmap

import "github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"

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
