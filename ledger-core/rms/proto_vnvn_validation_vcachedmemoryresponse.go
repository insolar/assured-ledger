package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func (m *VCachedMemoryResponse) Validate(currentPulse PulseNumber) error {
	if m.State.Reference.IsEmpty() {
		return throw.New("State.Reference should not be empty")
	}

	switch m.CallStatus {
	case CachedMemoryStateFound:
		if m.State.Class.IsEmpty() {
			return throw.New("State.Class should not be empty")
		}
	case CachedMemoryStateUnknown, CachedMemoryStateMissing:
		if !m.State.Class.IsEmpty() {
			return throw.New("State.Class should be empty")
		}
		if !m.State.Memory.IsEmpty() {
			return throw.New("State.Memory should be empty")
		}
	default:
		return throw.New("unexpected CallStatus")
	}

	return nil
}
