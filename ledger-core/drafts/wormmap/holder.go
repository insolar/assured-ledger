package wormmap

import "github.com/insolar/assured-ledger/ledger-core/vanilla/keyset"

type MapHolder interface {
	Len() int
	Get(Key) (interface{}, bool)
	Set(Key, interface{})
	EnumKeys(func(keyset.Key) bool) bool
	EnumEntries(func(keyset.Key, interface{}) bool) bool
}
