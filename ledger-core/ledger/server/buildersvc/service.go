// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package buildersvc

import (
<<<<<<< HEAD
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type StreamDropAssistant interface {
	CalculateJetDrop(reference.Holder) jet.DropID
	CreateJetDropAssistant(id jet.ID) JetDropAssistant
}

type JetDropAssistant interface {
	AddRecords(future *Future, br *lineage.BundleResolver) bool
	GetResolver() lineage.DependencyResolver
}

type Service interface {
	CreateStreamDrop(pulse.Range, /* jetTree, population */) (StreamDropAssistant, []jet.PrefixedID)
=======
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

type JetID uint32

type Service interface {
	CreateStreamDrop(pulse.Range, /* jetTree, population */) []JetID
>>>>>>> Ledger SMs
}

var _ Service = &serviceImpl{}

func NewService() Service {
	return &serviceImpl{}
}

type serviceImpl struct {}

<<<<<<< HEAD
func (p *serviceImpl) CreateStreamDrop(pulse.Range) (StreamDropAssistant, []jet.PrefixedID) {
=======
func (p *serviceImpl) CreateStreamDrop(pulse.Range) []JetID {
>>>>>>> Ledger SMs
	panic("implement me")
}


