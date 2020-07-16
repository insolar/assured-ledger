// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package member

// StaticRole holds role of node.
type StaticRole = PrimaryRole

const (
	StaticRoleUnknown = PrimaryRoleInactive
	StaticRoleVirtual = PrimaryRoleVirtual
	StaticRoleHeavyMaterial = PrimaryRoleHeavyMaterial
	StaticRoleLightMaterial = PrimaryRoleLightMaterial
)

// GetStaticRoleFromString converts role from string to StaticRole.
func GetStaticRoleFromString(role string) StaticRole {
	switch role {
	case "virtual":
		return StaticRoleVirtual
	case "heavy_material":
		return StaticRoleHeavyMaterial
	case "light_material":
		return StaticRoleLightMaterial
	}

	return StaticRoleUnknown
}
