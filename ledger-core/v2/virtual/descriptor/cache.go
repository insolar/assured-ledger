// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package descriptor

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor.Cache -o ./ -s _mock.go -g

// Cache provides convenient way to get prototype and code descriptors
// of objects without fetching them twice
type Cache interface {
	ByPrototypeRef(ctx context.Context, protoRef insolar.Reference) (PrototypeDescriptor, CodeDescriptor, error)
	ByObjectDescriptor(ctx context.Context, obj ObjectDescriptor) (PrototypeDescriptor, CodeDescriptor, error)
	GetPrototype(ctx context.Context, ref insolar.Reference) (PrototypeDescriptor, error)
	GetCode(ctx context.Context, ref insolar.Reference) (CodeDescriptor, error)
}
