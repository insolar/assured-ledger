package machinetype

// Type is a type of virtual machine
type Type int

// Real constants of Type
const (
	Unknown Type = iota
	Builtin
	LastID
)
