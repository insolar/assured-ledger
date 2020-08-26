// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package memorycache

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/virtual/memorycache.Service -o ./ -s _mock.go -g
type Service interface {
	Get(ctx context.Context, objectReference reference.Global) (descriptor.Object, error)
	Set(ctx context.Context, objectDescriptor descriptor.Object) (reference.Global, error)
}

type DefaultService struct {
	// todo
}

func (s *DefaultService) Get(ctx context.Context, objectReference reference.Global) (descriptor.Object, error) {
	// todo
	object := descriptor.NewObject(objectReference, reference.Local{}, reference.Global{}, []byte("implement me"), false)
	return object, nil
}

func (s *DefaultService) Set(ctx context.Context, objectDescriptor descriptor.Object) (reference.Global, error) {
	// todo
	return reference.Global{}, nil
}

func NewDefaultService() *DefaultService {
	return &DefaultService{}
}
