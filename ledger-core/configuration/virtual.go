package configuration

// Virtual holds configuration for virtual.
type Virtual struct {
	// MaxRunners limits number of contract executions running in parallel.
	// If set to zero or a negative value, limit will be set automatically to
	// `( runtime.NumCPU() - 2 ) but not less then 4`.
	MaxRunners int
}

// NewVirtual creates new default virtual configuration.
func NewVirtual() Virtual {
	return Virtual{
		MaxRunners: 0, // auto
	}
}
