// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package readersvc

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger/jetalloc"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/readersvc/readbundle"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ Service = &serviceImpl{}

func newService(allocStrategy jetalloc.MaterialAllocationStrategy, merklePair cryptkit.PairDigester, provider readbundle.Provider) *serviceImpl {
	switch {
	case allocStrategy == nil:
		panic(throw.IllegalValue())
	case merklePair == nil:
		panic(throw.IllegalValue())
	case provider == nil:
		panic(throw.IllegalValue())
	}
	return &serviceImpl{provider}
}

type serviceImpl struct {
	// allocStrategy jetalloc.MaterialAllocationStrategy
	// merklePair    cryptkit.PairDigester
	// mapMutex sync.RWMutex
	// lastPN   pulse.Number
	provider readbundle.Provider
}

func (p *serviceImpl) FindCabinet(pn pulse.Number) readbundle.ReadCabinet {
	return p.provider.FindCabinet(pn)
}

