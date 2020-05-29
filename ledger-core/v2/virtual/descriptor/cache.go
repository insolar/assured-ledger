// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package descriptor

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

type CacheCallbackType func(reference reference.Global) (interface{}, error)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor.Cache -o ./ -s _mock.go -g

// Cache provides convenient way to get class and code descriptors
// of objects without fetching them twice
type Cache interface {
	ByClassRef(ctx context.Context, classRef reference.Global) (Class, Code, error)
	RegisterCallback(cb CacheCallbackType)
}
