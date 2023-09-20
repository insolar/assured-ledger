package vermap

import "github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"

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

type txEntry struct {
	Entry
	tx *txMark
}
