package descriptor

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
)

// DescriptorsCache provides convenient way to get prototype and code descriptors
// of objects without fetching them twice
type DescriptorsCache interface {
	ByPrototypeRef(ctx context.Context, protoRef insolar.Reference) (PrototypeDescriptor, CodeDescriptor, error)
	ByObjectDescriptor(ctx context.Context, obj ObjectDescriptor) (PrototypeDescriptor, CodeDescriptor, error)
	GetPrototype(ctx context.Context, ref insolar.Reference) (PrototypeDescriptor, error)
	GetCode(ctx context.Context, ref insolar.Reference) (CodeDescriptor, error)
}
