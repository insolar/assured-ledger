package requestresult

type Type uint8

const (
	SideEffectNone Type = iota
	SideEffectActivate
	SideEffectAmend
	SideEffectDeactivate
)

func (t Type) String() string {
	switch t {
	case SideEffectNone:
		return "None"
	case SideEffectActivate:
		return "Activate"
	case SideEffectAmend:
		return "Amend"
	case SideEffectDeactivate:
		return "Deactivate"
	default:
		return "Unknown"
	}
}
