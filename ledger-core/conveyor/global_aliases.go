package conveyor

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
)

var _ smachine.SlotAliasRegistry = &GlobalAliases{}

type GlobalAliases struct {
	m sync.Map
}

func (p *GlobalAliases) UnpublishAlias(key interface{}) {
	p.m.Delete(key)
}

func (p *GlobalAliases) GetPublishedAlias(key interface{}) smachine.SlotAliasValue {
	if v, ok := p.m.Load(key); ok {
		if link, ok := v.(smachine.SlotAliasValue); ok {
			return link
		}
	}
	return smachine.SlotAliasValue{}
}

func (p *GlobalAliases) PublishAlias(key interface{}, slot smachine.SlotAliasValue) bool {
	_, loaded := p.m.LoadOrStore(key, slot)
	return !loaded
}

func (p *GlobalAliases) ReplaceAlias(key interface{}, slot smachine.SlotAliasValue) {
	p.m.Store(key, slot)
}
