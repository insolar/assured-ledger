// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package buildersvc

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type JetID uint32

type JetDropID uint64

func NewJetDropID(pn pulse.Number, id JetID) JetDropID {
	panic(throw.NotImplemented())
}

func (v JetDropID) IsValid() bool {
	panic(throw.NotImplemented())
}

func (v JetDropID) GetPulseNumber() pulse.Number {
	panic(throw.NotImplemented())
}


type StreamDropAssistant interface {
	CalculateJetDrop(reference.Holder) JetDropID
	CreateJetDropAssistant(id JetID) JetDropAssistant
}

type JetDropAssistant interface {
	JetDropAssistant()
	AddRecords(future *Future, br *lineage.BundleResolver) bool
	GetResolver() lineage.DependencyResolver
}


type Service interface {
	CreateStreamDrop(pulse.Range, /* jetTree, population */) (StreamDropAssistant, []JetID)
}

var _ Service = &serviceImpl{}

func NewService() Service {
	return &serviceImpl{}
}

type serviceImpl struct {}

func (p *serviceImpl) CreateStreamDrop(pulse.Range) (StreamDropAssistant, []JetID) {
	panic("implement me")
}


