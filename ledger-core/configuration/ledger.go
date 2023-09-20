package configuration

// Ledger holds configuration for ledger.
type Ledger struct {
	// LightChainLimit is maximum pulse difference (NOT number of pulses)
	// between current and the latest replicated on heavy.
	//
	// IMPORTANT: It should be the same on ALL nodes.
	// deprecated
	LightChainLimit int
}

// NewLedger creates new default Ledger configuration.
func NewLedger() Ledger {
	return Ledger{
		LightChainLimit: 5, // 5 pulses
	}
}
