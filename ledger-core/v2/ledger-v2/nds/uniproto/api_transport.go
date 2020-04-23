// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniproto

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/uniproto/l1"
)

type OutFunc func(l1.OutTransport) (canRetry bool, err error)
type OutTransport interface {
	UseSessionless(applyFn OutFunc) error
	UseSessionful(size int64, applyFn OutFunc) error
	UseAny(size int64, applyFn OutFunc) error
	EnsureConnect() error
}
