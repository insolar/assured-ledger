// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package server

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lmnapp"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/server/internal/virtual"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func appFactory(ctx context.Context, cfg configuration.Configuration, cmps insapp.AppComponents) (insapp.AppComponent, error) {
	switch cmps.Certificate.GetRole() {
	case member.PrimaryRoleVirtual:
		return virtual.AppFactory(ctx, cfg, cmps)
	case member.PrimaryRoleLightMaterial:
		return lmnapp.AppFactory(ctx, cfg, cmps)
	}
	panic(throw.IllegalValue())
}
