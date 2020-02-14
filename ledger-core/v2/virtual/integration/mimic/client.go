// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package mimic

import (
	"context"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/record"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger/artifact"
)

type client struct {
	storage Storage
}

func NewClient(storage Storage) artifact.Manager {
	return &client{storage: storage}
}

func (c *client) GetObject(ctx context.Context, head insolar.Reference) (artifact.ObjectDescriptor, error) {
	objectID := *head.GetLocal()
	state, index, _, err := c.storage.GetObject(ctx, objectID)
	if err != nil {
		return nil, err
	}

	return &objectDescriptor{
		head:        head,
		state:       *index.Lifeline.LatestState,
		prototype:   state.GetImage(),
		isPrototype: state.GetIsPrototype(),
		parent:      index.Lifeline.Parent,
		memory:      state.GetMemory(),
	}, nil
}

func (c *client) ActivateObject(
	ctx context.Context,
	domain, objectRef, parent, prototype insolar.Reference,
	memory []byte,
) error {
	objectID := *objectRef.GetLocal()

	mimicStorage := c.storage.(*mimicStorage)
	mimicStorage.Requests[objectID] = NewIncomingRequestEntity(objectID, &record.IncomingRequest{
		CallType: record.CTGenesis,
		Method:   objectRef.String(),
	})
	mimicStorage.Objects[objectID] = &ObjectEntity{
		ObjectChanges: nil,
		RequestsMap:   make(map[insolar.ID]*RequestEntity),
		RequestsList:  nil,
	}

	rec := record.Activate{
		Request:     objectRef,
		Memory:      memory,
		Image:       prototype,
		IsPrototype: false,
		Parent:      parent,
	}

	_, err := c.storage.SetResult(ctx, &record.Result{
		Request: objectRef,
		Payload: []byte{},
	})
	if err != nil {
		return errors.Wrap(err, "failed to store result")
	}

	return c.storage.SetObject(ctx, *objectRef.GetLocal(), &rec, *objectRef.GetLocal())
}

// FORCEFULLY DISABLED
func (c *client) RegisterRequest(ctx context.Context, req record.IncomingRequest) (*insolar.ID, error) {
	requestID := gen.ID()
	return &requestID, nil
}

// FORCEFULLY DISABLED
func (c *client) RegisterResult(ctx context.Context, obj, request insolar.Reference, payload []byte) (*insolar.ID, error) {
	return nil, nil
}

// NOT NEEDED
func (c client) UpdateObject(ctx context.Context, domain, request insolar.Reference, obj artifact.ObjectDescriptor, memory []byte) error {
	var (
		image *insolar.Reference
		err   error
	)
	if obj.IsPrototype() {
		image, err = obj.Code()
	} else {
		image, err = obj.Prototype()
	}
	if err != nil {
		return errors.Wrap(err, "failed to update object")
	}

	rec := record.Amend{
		Request:     request,
		Memory:      memory,
		Image:       *image,
		IsPrototype: obj.IsPrototype(),
		PrevState:   *obj.StateID(),
	}

	objectID := *obj.HeadRef().GetLocal()
	return c.storage.SetObject(ctx, insolar.ID{}, &rec, objectID)
}

// NOT NEEDED
func (c client) DeployCode(
	ctx context.Context,
	domain insolar.Reference,
	request insolar.Reference,
	code []byte,
	machineType insolar.MachineType,
) (*insolar.ID, error) {
	panic("implement me")
}
