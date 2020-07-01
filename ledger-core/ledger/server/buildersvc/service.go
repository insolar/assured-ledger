// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package buildersvc

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

type JetID uint32

type Service interface {
	CreateStreamDrop(pulse.Range, /* jetTree, population */) []JetID
}

var _ Service = &serviceImpl{}

func NewService() Service {
	return &serviceImpl{}
}

type serviceImpl struct {}

func (p *serviceImpl) CreateStreamDrop(pulse.Range) []JetID {
	panic("implement me")
}


