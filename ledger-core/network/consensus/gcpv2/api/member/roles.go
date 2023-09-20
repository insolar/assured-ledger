package member

type PrimaryRole uint8 // MUST BE 6-bit

const (
	PrimaryRoleInactive PrimaryRole = iota
	PrimaryRoleNeutral
	PrimaryRoleHeavyMaterial
	PrimaryRoleLightMaterial
	PrimaryRoleVirtual
	// PrimaryRoleCascade
	// PrimaryRoleRecrypt
	PrimaryRoleCount = iota

    PrimaryRoleUnknown = PrimaryRoleInactive
)

func (v PrimaryRole) Equal(other PrimaryRole) bool {
	return v == other
}

func (v PrimaryRole) IsMaterial() bool {
	return v == PrimaryRoleHeavyMaterial || v == PrimaryRoleLightMaterial
}

func (v PrimaryRole) IsHeavyMaterial() bool {
	return v == PrimaryRoleHeavyMaterial
}

func (v PrimaryRole) IsLightMaterial() bool {
	return v == PrimaryRoleLightMaterial
}

func (v PrimaryRole) IsVirtual() bool {
	return v == PrimaryRoleVirtual
}

func (v PrimaryRole) IsNeutral() bool {
	return v == PrimaryRoleNeutral
}

func (v PrimaryRole) String() string {
	switch v {
	case PrimaryRoleVirtual:
		return "virtual"
	case PrimaryRoleHeavyMaterial:
		return "heavy_material"
	case PrimaryRoleLightMaterial:
		return "light_material"
	case PrimaryRoleNeutral:
		return "neutral"
	case PrimaryRoleInactive:
		return "inactive"
	}

	return "unknown"
}

// GetPrimaryRoleFromString converts role from string to PrimaryRole.
func GetPrimaryRoleFromString(role string) PrimaryRole {
	switch role {
	case "virtual":
		return PrimaryRoleVirtual
	case "heavy_material":
		return PrimaryRoleHeavyMaterial
	case "light_material":
		return PrimaryRoleLightMaterial
	}

	return PrimaryRoleUnknown
}

type SpecialRole uint8

const (
	SpecialRoleNone      SpecialRole = 0
	SpecialRoleDiscovery SpecialRole = 1 << iota
)

func (v SpecialRole) IsDiscovery() bool {
	return v == SpecialRoleDiscovery
}

func (v SpecialRole) Equal(other SpecialRole) bool {
	return v == other
}
