// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"fmt"
	"strings"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type SynchronizationContext interface {
	// Check provides current state of a sync object.
	// When the sync was previously acquired, then this function returns SM's status of a sync object.
	// When the sync was not previously acquired, then this function returns a general status of a sync object
	// Panics on zero or incorrectly initialized value.
	Check(SyncLink) BoolDecision

	// Acquire acquires a holder of the sync object and returns status of the acquired holder:
	//
	// 1) Passed/true - SM can proceed to access resources controlled by this sync object.
	//    Passed holder MUST be released to ensure that other SM can also pass.
	//
	// 2) NotPassed/false - SM can't proceed to access resources controlled by this sync object.
	//    NotPassed holder remains valid and ensures that SM retains location an a queue of the sync object.
	//    NotPassed holder will at some moment be converted into Passed holder and the relevant SM will be be woken up.
	//    NotPassed holder MUST be released.
	//
	// Acquired holder will be released when SM is stopped.
	// Panics on zero or incorrectly initialized value.
	// Panics when another sync was acquired, but was not released. This also applies then holder was previously acquired with different methods/flags/priority.
	Acquire(SyncLink) BoolDecision
	// AcquireAndRelease releases any previously acquired sync object AFTER acquiring a new one.
	AcquireAndRelease(SyncLink) BoolDecision

	// AcquireForThisStep is similar to Acquire(), but the acquired holder will also be released when step is changed.
	// To avoid doubt - Repeat(), WakeUp() and Stay() operations will not release.
	// Other operations, including Jump() to the same step will do RELEASE even if they combined with Sleep() or similar predicates.
	// Panics on zero or incorrectly initialized value.
	AcquireForThisStep(SyncLink) BoolDecision
	// AcquireForThisStepAndRelease combines features of both AcquireForThisStep() and AcquireAndRelease()
	AcquireForThisStepAndRelease(SyncLink) BoolDecision

	// AcquireExt acquires a holder of the sync object with additional features/properties defined by AcquireFlags. See also Acquire* methods.
	AcquireExt(SyncLink, AcquireFlags) BoolDecision

	// Release releases a holder of this SM for the given sync object.
	// When there is no holder or the current holder belongs to a different sync object then operation is ignored and false is returned.
	// NB! Some sync objects (e.g. conditionals) may release a passed holder automatically, hence this function will return false as well.
	// Panics on zero or incorrectly initialized value.
	Release(SyncLink) bool

	minimalSynchronizationContext
}

type AcquireFlags uint8

const (
	// AcquireForThisStep flag grants property of SynchronizationContext.AcquireForThisStep()
	AcquireForThisStep AcquireFlags = 1<<iota
	// AcquireAndRelease flag grants property of SynchronizationContext.AcquireAndRelease()
	AcquireAndRelease
	// BoostedPriorityAcquire flag grants property of a boosted slot - it gets higher priority on relevant synchronization queues.
	BoostedPriorityAcquire
	// HighPriorityAcquire flag grants property of a priority slot - it gets the highest priority on relevant synchronization queues.
	// USE WITH CAUTION.
	HighPriorityAcquire
	// NoPriorityAcquire flag suppresses slot's boost/priority status for the acquire operation.
	NoPriorityAcquire
)

type minimalSynchronizationContext interface {
	// Releases a holder of this SM for any sync object if present.
	// Returns true when a holder of a sync object was released.
	// NB! Some sync objects (e.g. conditionals) may release a passed holder automatically, hence this function will return false as well.
	// Panics on zero or incorrectly initialized value.
	ReleaseAll() bool

	// Applies the given adjustment to a relevant sync object. SM doesn't need to acquire the relevant sync object.
	// Returns true when at least one holder of the sync object was affected.
	// Panics on zero or incorrectly initialized value.
	ApplyAdjustment(SyncAdjustment) bool
}


func NewSyncLink(controller DependencyController) SyncLink {
	if controller == nil {
		panic(throw.IllegalValue())
	}
	return SyncLink{controller}
}

// Represents a sync object.
type SyncLink struct {
	controller DependencyController
}

func (v SyncLink) IsZero() bool {
	return v.controller == nil
}

// Provides an implementation depended state of the sync object.
// Safe for concurrent use.
func (v SyncLink) GetCounts() (active, inactive int) {
	return v.controller.GetCounts()
}

// Provides an implementation depended state of the sync object
// Safe for concurrent use.
func (v SyncLink) GetLimit() (limit int, isAdjustable bool) {
	return v.controller.GetLimit()
}

func (v SyncLink) DebugPrint(maxCount int) {
	limit, _ := v.GetLimit()
	active, inactive := v.GetCounts()
	fmt.Printf("%s[l=%d, a=%d, i=%d] {", v.String(), limit, active, inactive)

	lastQ := 0
	hasQ := false
	lastM := ""
	v.controller.EnumQueues(func(qId int, link SlotLink, _ SlotDependencyFlags) bool {
		maxCount--
		prefix := ""
		switch {
		case maxCount < 0:
			fmt.Print(", ...")
			return true
		case lastQ != qId || !hasQ:
			lastQ = qId
			if hasQ {
				fmt.Printf("} Q#%d{", qId)
			} else {
				hasQ = true
				fmt.Printf(" Q#%d{", qId)
			}
		default:
			prefix = ", "
		}
		mPrefix := link.MachineID()
		if lastM != mPrefix {
			lastM = mPrefix
			fmt.Print(prefix, "M#", mPrefix, ":", link.SlotID())
		} else {
			fmt.Print(prefix, link.SlotID())
		}
		return false
	})
	fmt.Println("}")
}

func (v SyncLink) String() string {
	name := v.controller.GetName()
	if len(name) > 0 {
		return name
	}
	return fmt.Sprintf("sync-%p", v.controller)
}

/* ============================================== */

func NewSyncAdjustment(controller DependencyController, adjustment int, isAbsolute bool) SyncAdjustment {
	if controller == nil {
		panic(throw.IllegalValue())
	}
	return SyncAdjustment{controller, adjustment, isAbsolute}
}

type SyncAdjustment struct {
	controller DependencyController
	adjustment int
	isAbsolute bool
}

func (v SyncAdjustment) IsZero() bool {
	return v.controller == nil
}

func (v SyncAdjustment) IsEmpty() bool {
	return v.controller == nil || !v.isAbsolute && v.adjustment == 0
}

func (v SyncAdjustment) MergeWith(u ...SyncAdjustment) SyncAdjustment {
	if v.controller == nil {
		panic(throw.IllegalState())
	}

	total := 0
	for _, adj := range u {
		if adj.IsZero() {
			panic(throw.IllegalValue())
		}
		if ma, ok := adj.controller.(mergedAdjustments); ok {
			total += len(ma.adjustments)
		} else {
			total++
		}
	}

	if total == 0 {
		return v
	}

	tma := mergedAdjustments{ make([]SyncAdjustment, 1, 1 + total) }
	tma.adjustments[0] = v

	for _, adj := range u {
		if ma, ok := adj.controller.(mergedAdjustments); ok {
			tma.adjustments = append(tma.adjustments, ma.adjustments...)
		} else {
			tma.adjustments = append(tma.adjustments, adj)
		}
	}

	return SyncAdjustment{ tma, 0, false }
}

func (v SyncAdjustment) String() string {
	if _, ok := v.controller.(mergedAdjustments); ok {
		return v.controller.GetName()
	}

	name := SyncLink{v.controller}.String()
	switch {
	case v.isAbsolute:
		return fmt.Sprintf("%s[=%d]", name, v.adjustment)
	case v.adjustment < 0:
		return fmt.Sprintf("%s[%d]", name, v.adjustment)
	default:
		return fmt.Sprintf("%s[+%d]", name, v.adjustment)
	}
}

/* ============================================== */

type SlotDependencyFlags uint8

const (
	SyncPriorityBoosted SlotDependencyFlags = 1 << iota
	SyncPriorityHigh
	SyncForOneStep
	SyncIgnoreFlags
)

const SyncPriorityMask = SyncPriorityBoosted | SyncPriorityHigh

func (v SlotDependencyFlags) HasLessPriorityThan(o SlotDependencyFlags) bool {
	return v&SyncPriorityMask < o&SyncPriorityMask
}

func (v SlotDependencyFlags) IsCompatibleWith(requiredFlags SlotDependencyFlags) bool {
	if requiredFlags == SyncIgnoreFlags {
		return true
	}

	if v&requiredFlags&^SyncPriorityMask != requiredFlags&^SyncPriorityMask {
		return false
	}
	return !v.HasLessPriorityThan(requiredFlags)
}

type EnumQueueFunc func(qId int, link SlotLink, flags SlotDependencyFlags) bool

// Internals of a sync object
type DependencyController interface {
	// CheckState returns current state (open = true, closed = false)
	CheckState() BoolDecision
	// CreateDependency creates a dependency to this sync object.
	// Can return (true, nil) when this sync object doesn't have "open" limit, e.g. for conditional sync.
	CreateDependency(holder SlotLink, flags SlotDependencyFlags) (BoolDecision, SlotDependency)
	// UseDependency also handles partial acquire of hierarchical syncs
	UseDependency(dep SlotDependency, flags SlotDependencyFlags) Decision
	// ReleaseDependency does partial release of hierarchical syncs. MUST be called only after UseDependency check
	ReleaseDependency(dep SlotDependency) (bool, SlotDependency, []PostponedDependency, []StepLink)

	GetLimit() (limit int, isAdjustable bool)
	AdjustLimit(limit int, absolute bool) (deps []StepLink, activate bool)

	GetCounts() (active, inactive int)
	GetName() string

	EnumQueues(EnumQueueFunc) bool
}

/****************************************/

var _ DependencyController = mergedAdjustments{}
// mergedAdjustments is a stub controller to merge multiple adjustments
type mergedAdjustments struct {
	adjustments []SyncAdjustment
}

func (v mergedAdjustments) AdjustLimit(int, bool) (deps []StepLink, activate bool) {
	for _, adj := range v.adjustments {
		switch adjDeps, adjAct := adj.controller.AdjustLimit(adj.adjustment, adj.isAbsolute); {
		case adjAct:
			activate = true
			deps = append(deps, adjDeps...)
		case len(adjDeps) > 0:
			panic(throw.Impossible())
		}
	}
	return
}

func (v mergedAdjustments) GetName() string {
	sb := strings.Builder{}
	for i, adj := range v.adjustments {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(adj.String())
	}
	return sb.String()
}

func (v mergedAdjustments) CheckState() BoolDecision { panic(throw.Unsupported()) }
func (v mergedAdjustments) CreateDependency(SlotLink, SlotDependencyFlags) (BoolDecision, SlotDependency) { panic(throw.Unsupported()) }
func (v mergedAdjustments) UseDependency(SlotDependency, SlotDependencyFlags) Decision { panic(throw.Unsupported()) }
func (v mergedAdjustments) ReleaseDependency(dep SlotDependency) (bool, SlotDependency, []PostponedDependency, []StepLink) { panic(throw.Unsupported()) }
func (v mergedAdjustments) GetLimit() (int, bool) { panic(throw.Unsupported()) }
func (v mergedAdjustments) GetCounts() (int, int) { panic(throw.Unsupported()) }
func (v mergedAdjustments) EnumQueues(queueFunc EnumQueueFunc) bool { panic(throw.Unsupported()) }

