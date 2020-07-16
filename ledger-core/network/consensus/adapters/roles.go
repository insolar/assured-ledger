// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package adapters

import (
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
)

func StaticRoleToPrimaryRole(staticRole member.StaticRole) member.PrimaryRole {
	switch staticRole {
	case member.StaticRoleVirtual:
		return member.PrimaryRoleVirtual
	case member.StaticRoleLightMaterial:
		return member.PrimaryRoleLightMaterial
	case member.StaticRoleHeavyMaterial:
		return member.PrimaryRoleHeavyMaterial
	case member.StaticRoleUnknown:
		fallthrough
	default:
		return member.PrimaryRoleNeutral
	}
}

func PrimaryRoleToStaticRole(primaryRole member.PrimaryRole) member.StaticRole {
	switch primaryRole {
	case member.PrimaryRoleVirtual:
		return member.StaticRoleVirtual
	case member.PrimaryRoleLightMaterial:
		return member.StaticRoleLightMaterial
	case member.PrimaryRoleHeavyMaterial:
		return member.StaticRoleHeavyMaterial
	case member.PrimaryRoleNeutral:
		fallthrough
	default:
		return member.StaticRoleUnknown
	}
}
