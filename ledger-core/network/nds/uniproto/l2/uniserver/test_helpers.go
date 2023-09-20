package uniserver

import (
	"crypto/tls"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l1"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

type FuncMapSessionlessProvider func(l1.SessionlessTransportProvider) l1.SessionlessTransportProvider
type FuncMapSessionfullProvider func(l1.SessionfulTransportProvider) l1.SessionfulTransportProvider

func MapTransportProvider(
	provider AllTransportProvider,
	mappingLessFunc FuncMapSessionlessProvider,
	mappingFullFunc FuncMapSessionfullProvider,
) AllTransportProvider {
	return &MappedTransportProvider{
		AllTransportProvider: provider,
		mappingLessFunc:      mappingLessFunc,
		mappingFullFunc:      mappingFullFunc,
	}
}

type MappedTransportProvider struct {
	AllTransportProvider
	mappingLessFunc FuncMapSessionlessProvider
	mappingFullFunc FuncMapSessionfullProvider
}

func (p *MappedTransportProvider) CreateSessionlessProvider(binding nwapi.Address, preference nwapi.Preference, maxUDPSize uint16) l1.SessionlessTransportProvider {
	sessionlessProvider := p.AllTransportProvider.CreateSessionlessProvider(binding, preference, maxUDPSize)

	if p.mappingLessFunc != nil {
		return p.mappingLessFunc(sessionlessProvider)
	}
	return sessionlessProvider
}

func (p *MappedTransportProvider) CreateSessionfulProvider(binding nwapi.Address, preference nwapi.Preference, tlsCfg *tls.Config) l1.SessionfulTransportProvider {
	sessionfullProvider := p.AllTransportProvider.CreateSessionfulProvider(binding, preference, tlsCfg)

	if p.mappingFullFunc != nil {
		return p.mappingFullFunc(sessionfullProvider)
	}
	return sessionfullProvider
}
