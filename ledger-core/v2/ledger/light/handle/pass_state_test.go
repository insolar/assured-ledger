// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package handle_test

import (
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/flow"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger/light/handle"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger/light/proc"
)

func TestPassState_Present(t *testing.T) {
	ctx := inslogger.TestContext(t)
	mc := minimock.NewController(t)

	var (
		dep  *proc.Dependencies
		meta payload.Meta
	)

	setup := func() {
		dep = proc.NewDependenciesMock()
	}

	t.Run("basic ok", func(t *testing.T) {
		setup()
		defer mc.Finish()

		getObject, _ := (&payload.GetObject{
			Polymorph: uint32(payload.TypeGetObject),
			ObjectID:  gen.ID(),
		}).Marshal()
		originMeta, _ := (&payload.Meta{
			Polymorph: uint32(payload.TypeMeta),
			Payload:   getObject,
		}).Marshal()

		passState, _ := (&payload.PassState{
			Polymorph: uint32(payload.TypePassState),
			Origin:    originMeta,
			StateID:   gen.ID(),
		}).Marshal()

		meta = payload.Meta{
			Polymorph: uint32(payload.TypeMeta),
			Payload:   passState,
		}

		handler := handle.NewPassState(dep, meta)
		flowMock := flow.NewFlowMock(mc).ProcedureMock.Return(nil)
		err := handler.Present(ctx, flowMock)
		assert.NoError(t, err)
	})

	t.Run("PassState procedure returns err", func(t *testing.T) {
		setup()
		defer mc.Finish()

		getObject, _ := (&payload.GetObject{
			Polymorph: uint32(payload.TypeGetObject),
			ObjectID:  gen.ID(),
		}).Marshal()
		originMeta, _ := (&payload.Meta{
			Polymorph: uint32(payload.TypeMeta),
			Payload:   getObject,
		}).Marshal()

		passState, _ := (&payload.PassState{
			Polymorph: uint32(payload.TypePassState),
			Origin:    originMeta,
			StateID:   gen.ID(),
		}).Marshal()

		meta = payload.Meta{
			Polymorph: uint32(payload.TypeMeta),
			Payload:   passState,
		}

		handler := handle.NewPassState(dep, meta)
		flowMock := flow.NewFlowMock(mc).ProcedureMock.Return(errors.New("error from PassState"))
		err := handler.Present(ctx, flowMock)
		assert.EqualError(t, err, "error from PassState")
	})

	t.Run("Message type is wrong returns error", func(t *testing.T) {
		setup()
		defer mc.Finish()

		getObject, _ := (&payload.Activate{
			Polymorph: uint32(payload.TypeActivate),
		}).Marshal()
		originMeta, _ := (&payload.Meta{
			Polymorph: uint32(payload.TypeMeta),
			Payload:   getObject,
		}).Marshal()

		passState, _ := (&payload.PassState{
			Polymorph: uint32(payload.TypePassState),
			Origin:    originMeta,
			StateID:   gen.ID(),
		}).Marshal()

		meta = payload.Meta{
			Polymorph: uint32(payload.TypeMeta),
			Payload:   passState,
		}

		handler := handle.NewPassState(dep, meta)
		flowMock := flow.NewFlowMock(mc)
		err := handler.Present(ctx, flowMock)
		assert.EqualError(t, err, "unexpected payload type *payload.Activate")
	})
}
