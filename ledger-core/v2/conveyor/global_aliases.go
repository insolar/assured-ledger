// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package conveyor

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
)

var _ smachine.SlotAliasRegistry = &GlobalAliases{}

type GlobalAliases struct {
	m sync.Map
}

func (p *GlobalAliases) UnpublishAlias(key interface{}) {
	p.m.Delete(key)
}

func (p *GlobalAliases) GetPublishedAlias(key interface{}) smachine.SlotLink {
	if v, ok := p.m.Load(key); ok {
		if link, ok := v.(smachine.SlotLink); ok {
			return link
		}
	}
	return smachine.SlotLink{}
}

func (p *GlobalAliases) PublishAlias(key interface{}, slot smachine.SlotLink) bool {
	_, loaded := p.m.LoadOrStore(key, slot)
	return !loaded
}
