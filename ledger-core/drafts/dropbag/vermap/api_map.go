package vermap

import "github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"

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
