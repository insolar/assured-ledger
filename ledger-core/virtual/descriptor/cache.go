package descriptor

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type CacheCallbackType func(reference reference.Global) (interface{}, error)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/virtual/descriptor.Cache -o ./ -s _mock.go -g

// Cache provides convenient way to get class and code descriptors
// of objects without fetching them twice
type Cache interface {
	ByClassRef(ctx context.Context, classRef reference.Global) (Class, Code, error)
	RegisterCallback(cb CacheCallbackType)
}
