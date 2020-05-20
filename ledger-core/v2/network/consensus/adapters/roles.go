// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package adapters

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/gcpv2/api/member"
)

func StaticRoleToPrimaryRole(staticRole node.StaticRole) member.PrimaryRole {
	switch staticRole {
	case node.StaticRoleVirtual:
		return member.PrimaryRoleVirtual
	case node.StaticRoleLightMaterial:
		return member.PrimaryRoleLightMaterial
	case node.StaticRoleHeavyMaterial:
		return member.PrimaryRoleHeavyMaterial
	case node.StaticRoleUnknown:
		fallthrough
	default:
		return member.PrimaryRoleNeutral
	}
}

func PrimaryRoleToStaticRole(primaryRole member.PrimaryRole) node.StaticRole {
	switch primaryRole {
	case member.PrimaryRoleVirtual:
		return node.StaticRoleVirtual
	case member.PrimaryRoleLightMaterial:
		return node.StaticRoleLightMaterial
	case member.PrimaryRoleHeavyMaterial:
		return node.StaticRoleHeavyMaterial
	case member.PrimaryRoleNeutral:
		fallthrough
	default:
		return node.StaticRoleUnknown
	}
}
