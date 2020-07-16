// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package adapters

import (
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
)

func StaticRoleToPrimaryRole(staticRole member.PrimaryRole) member.PrimaryRole {
	switch staticRole {
	case member.PrimaryRoleVirtual:
		return member.PrimaryRoleVirtual
	case member.PrimaryRoleLightMaterial:
		return member.PrimaryRoleLightMaterial
	case member.PrimaryRoleHeavyMaterial:
		return member.PrimaryRoleHeavyMaterial
	case member.PrimaryRoleUnknown:
		fallthrough
	default:
		return member.PrimaryRoleNeutral
	}
}

func PrimaryRoleToStaticRole(primaryRole member.PrimaryRole) member.PrimaryRole {
	switch primaryRole {
	case member.PrimaryRoleVirtual:
		return member.PrimaryRoleVirtual
	case member.PrimaryRoleLightMaterial:
		return member.PrimaryRoleLightMaterial
	case member.PrimaryRoleHeavyMaterial:
		return member.PrimaryRoleHeavyMaterial
	case member.PrimaryRoleNeutral:
		fallthrough
	default:
		return member.PrimaryRoleUnknown
	}
}
