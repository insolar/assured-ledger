// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package buildersvc

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type JetID uint32

type JetDropID uint64

func NewJetDropID(pn pulse.Number, id JetID) JetDropID {

}

func (v JetDropID) IsValid() bool {

}

type StreamDropAssistant interface {
	CalculateJetDrop(reference.Holder) JetDropID
	CreateJetDropAssistant(id JetID) JetDropAssistant
}

type JetDropAssistant interface {
	JetDropAssistant()
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


