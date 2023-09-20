package smachine

import (
	"fmt"
	"reflect"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type SharedDataFunc func(interface{}) (wakeup bool)

func NewUnboundSharedData(data interface{}) SharedDataLink {
	if data == nil {
		panic(throw.IllegalValue())
	}

	return SharedDataLink{
		data:  data,
		flags: ShareDataUnbound,
	}
}

// SharedDataLink enables data sharing between different slots.
type SharedDataLink struct {
	link  SlotLink
	data  interface{}
	flags ShareDataFlags
}

func (v SharedDataLink) IsZero() bool {
	return v.data == nil
}

// IsValid is ture when SharedDataLink is initialized and the data is valid at the moment of this call.
func (v SharedDataLink) IsValid() bool {
	return !v.IsZero() && (v.link.s == nil || v.link.IsValid())
}

// IsUnbound is true when this link points to an unbound data. Such data is always available, but must be safe for concurrent use.
func (v SharedDataLink) IsUnbound() bool {
	return v.link.s == nil
}

// IsDirectAccess returns true when data is either unbound, or shared as direct.
func (v SharedDataLink) IsDirectAccess() bool {
	return v.IsUnbound() || v.isDirect()
}

func (v SharedDataLink) isDirect() bool {
	return v.flags&ShareDataDirect != 0
}

func (v SharedDataLink) isSharedForOthers() bool {
	return v.flags&(ShareDataDirect|ShareDataUnbound|ShareDataWithOtherSlotMachines) != 0
}

func (v SharedDataLink) isOwnedBy(local *Slot) bool {
	return v.link.s == nil || v.link.s == local
}

func (v SharedDataLink) getData() interface{} {
	_, d := v.getDataAndMachine()
	return d
}

func (v SharedDataLink) getDataAndMachine() (*SlotMachine, interface{}) {
	m := v.link.getActiveMachine()
	if _, ok := v.data.(*uniqueSharedKey); ok {
		if v.IsDirectAccess() { // shouldn't happen
			panic(throw.Impossible())
		}
		if m != nil {
			if data, ok := m.localRegistry.Load(v.data); ok {
				return m, data
			}
		}
		return nil, nil
	}
	return m, v.data
}

// IsOfType returns true when the underlying data is of the given type
func (v SharedDataLink) IsOfType(t reflect.Type) bool {
	if a, ok := v.data.(*uniqueSharedKey); ok {
		return a.valueType == t
	}
	return reflect.TypeOf(v.data) == t
}

// IsAssignableToType returns true when the underlying data can be assigned to the given type
func (v SharedDataLink) IsAssignableToType(t reflect.Type) bool {
	switch a := v.data.(type) {
	case nil:
		return false
	case *uniqueSharedKey:
		return a.valueType.AssignableTo(t)
	}
	return reflect.TypeOf(v.data).AssignableTo(t)
}

// IsAssignableTo returns true when the underlying data can be assigned to the given value
func (v SharedDataLink) IsAssignableTo(t interface{}) bool {
	switch a := v.data.(type) {
	case nil:
		return false
	case *uniqueSharedKey:
		return a.valueType.AssignableTo(reflect.TypeOf(t))
	}
	return reflect.TypeOf(v.data).AssignableTo(reflect.TypeOf(t))
}

// EnsureType panics when the underlying data is of a different type
func (v SharedDataLink) EnsureType(t reflect.Type) {
	if v.data == nil {
		panic(throw.IllegalState())
	}
	dt := reflect.TypeOf(v.data)
	if !dt.AssignableTo(t) {
		panic(fmt.Sprintf("type mismatch: actual=%v expected=%v", dt, t))
	}
}

// TryDirectAccess returns related data from an unbound or from a valid direct link. Returns nil otherwise.
func (v SharedDataLink) TryDirectAccess() interface{} {
	switch {
	case v.data == nil:
		panic(throw.IllegalState())
	case v.IsUnbound():
		return v.data
	case v.isDirect():
		if v.link.IsValid() {
			return v.data
		}
	}
	return nil
}

// PrepareAccess creates an accessor that will apply the given function to the shared data.
// SharedDataAccessor gets same data retention rules as the original SharedDataLink.
func (v SharedDataLink) PrepareAccess(fn SharedDataFunc) SharedDataAccessor {
	if fn == nil {
		panic("illegal value")
	}
	return SharedDataAccessor{v, fn}
}

// SharedDataAccessor represents SharedDataLink with an accessor function attached.
type SharedDataAccessor struct {
	link     SharedDataLink
	accessFn SharedDataFunc
}

func (v SharedDataAccessor) IsZero() bool {
	return v.link.IsZero()
}

// TryUse is a convenience wrapper of ExecutionContext.UseShared()
func (v SharedDataAccessor) TryUse(ctx ExecutionContext) SharedAccessReport {
	return ctx.UseShared(v)
}

// TryUseDirectAccess is a convenience equivalent of TryUse, but it can access unbound or direct values.
func (v SharedDataAccessor) TryUseDirectAccess() SharedAccessReport {
	switch v.accessByOwner(nil) {
	case Passed:
		return SharedSlotAvailableAlways
	case Impossible:
		return SharedSlotAbsent
	}
	return SharedSlotRemoteBusy
}

func (v SharedDataAccessor) accessByOwner(local *Slot) Decision {
	if v.accessFn == nil || v.link.IsZero() {
		return Impossible
	}
	if !v.link.isOwnedBy(local) {
		return NotPassed
	}

	data := v.link.getData()
	if data == nil {
		return Impossible
	}
	v.accessFn(data)
	return Passed
}

var _ Decider = SharedAccessReport(0)

// Describes a result of shared data access
type SharedAccessReport uint8

const (
	// SharedSlotAbsent indicates that a link was invalidated or is inaccessible and can't be accessed anytime further.
	SharedSlotAbsent SharedAccessReport = iota
	// SharedSlotLocalBusy indicates that the link is valid, but data is in use by someone else. Data is from the same SlotMachine.
	SharedSlotLocalBusy
	// SharedSlotRemoteBusy indicates that the link is valid, but data is in use by someone else. Data is from a different SlotMachine.
	SharedSlotRemoteBusy
	// SharedSlotAvailableAlways indicates that the data is accessible or was accessed. Data is always available: is unbound, direct or belongs to this slot.
	SharedSlotAvailableAlways
	// SharedSlotLocalAvailable indicates that the data is accessible or was accessed. Data is from the same SlotMachine.
	SharedSlotLocalAvailable
	// SharedSlotRemoteAvailable indicates that the data is accessible or was accessed. Data is from a different SlotMachine.
	SharedSlotRemoteAvailable
)

func (v SharedAccessReport) IsAvailable() bool {
	return v >= SharedSlotAvailableAlways
}

func (v SharedAccessReport) IsRemote() bool {
	return v == SharedSlotRemoteBusy || v == SharedSlotRemoteAvailable
}

func (v SharedAccessReport) IsAbsent() bool {
	return v == SharedSlotAbsent
}

func (v SharedAccessReport) IsBusy() bool {
	return v == SharedSlotLocalBusy || v == SharedSlotRemoteBusy
}

func (v SharedAccessReport) GetDecision() Decision {
	switch {
	case v.IsAvailable():
		return Passed
	case v.IsAbsent():
		return Impossible
	default:
		return NotPassed
	}
}
