// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package wormmap

import "github.com/insolar/assured-ledger/ledger-core/vanilla/keyset"

type MapHolder interface {
	Len() int
	Get(Key) (interface{}, bool)
	Set(Key, interface{})
	EnumKeys(func(keyset.Key) bool) bool
	EnumEntries(func(keyset.Key, interface{}) bool) bool
}
