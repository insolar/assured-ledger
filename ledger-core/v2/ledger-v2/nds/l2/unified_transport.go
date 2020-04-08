// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l2

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/l1"
)

type UnifiedOutTransport interface {
	UseSessionless(canRetry bool, applyFn func(l1.OutTransport) error) error
	UseSessionful(size int64, canRetry bool, applyFn func(l1.OutTransport) error) error
	UseAny(size int64, canRetry bool, applyFn func(l1.OutTransport) error) error
	EnsureConnect() error
}
