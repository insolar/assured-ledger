// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package inspectsvc

type Service interface {
	InspectRecordSet(RegisterRequestSet) (InspectedRecordSet, error)
}

var _ Service = &serviceImpl{}

func NewService() Service {
	return &serviceImpl{}
}

type serviceImpl struct {}

func (p *serviceImpl) InspectRecordSet(set RegisterRequestSet) (InspectedRecordSet, error) {
	panic("implement me")
}
