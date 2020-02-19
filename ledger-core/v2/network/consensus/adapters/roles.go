// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package adapters

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/gcpv2/api/member"
)

func StaticRoleToPrimaryRole(staticRole insolar.StaticRole) member.PrimaryRole {
	switch staticRole {
	case insolar.StaticRoleVirtual:
		return member.PrimaryRoleVirtual
	case insolar.StaticRoleLightMaterial:
		return member.PrimaryRoleLightMaterial
	case insolar.StaticRoleHeavyMaterial:
		return member.PrimaryRoleHeavyMaterial
	case insolar.StaticRoleUnknown:
		fallthrough
	default:
		return member.PrimaryRoleNeutral
	}
}

func PrimaryRoleToStaticRole(primaryRole member.PrimaryRole) insolar.StaticRole {
	switch primaryRole {
	case member.PrimaryRoleVirtual:
		return insolar.StaticRoleVirtual
	case member.PrimaryRoleLightMaterial:
		return insolar.StaticRoleLightMaterial
	case member.PrimaryRoleHeavyMaterial:
		return insolar.StaticRoleHeavyMaterial
	case member.PrimaryRoleNeutral:
		fallthrough
	default:
		return insolar.StaticRoleUnknown
	}
}
