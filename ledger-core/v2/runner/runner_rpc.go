// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/insolar/blob/master/LICENSE.md.

package runner

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executor/common/rpctypes"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

func (r *DefaultService) GetCode(_ rpctypes.UpGetCodeReq, _ *rpctypes.UpGetCodeResp) error {
	panic(throw.NotImplemented())
}

func (r *DefaultService) RouteCall(_ rpctypes.UpRouteReq, _ *rpctypes.UpRouteResp) error {
	panic(throw.NotImplemented())
}

func (r *DefaultService) SaveAsChild(_ rpctypes.UpSaveAsChildReq, _ *rpctypes.UpSaveAsChildResp) error {
	panic(throw.NotImplemented())
}

func (r *DefaultService) DeactivateObject(_ rpctypes.UpDeactivateObjectReq, _ *rpctypes.UpDeactivateObjectResp) error {
	panic(throw.NotImplemented())
}
