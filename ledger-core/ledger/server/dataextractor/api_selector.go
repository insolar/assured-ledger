package dataextractor

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type Direction bool

const (
	ToPresent = false
	ToPast = true
)

//nolint:gosimple
func (v Direction) IsToPast() bool {
	return v == ToPast
}

type Selector struct {
	RootRef   reference.Holder
	ReasonRef reference.Holder
	StartRef  reference.Holder
	StopRef   reference.Holder

	BranchMode BranchMode
	Direction  Direction
}

func (v Selector) GetSelectorRef() (bool, reference.Holder) {
	switch {
	case !reference.IsEmpty(v.StartRef):
		return true, v.StartRef
	case !reference.IsEmpty(v.StopRef):
		return true, v.StopRef
	case !reference.IsEmpty(v.RootRef):
		return true, v.RootRef
	case !reference.IsEmpty(v.ReasonRef):
		return false, v.ReasonRef
	default:
		panic(throw.IllegalState())
	}
}

type BranchMode uint8

const (
	_ BranchMode = iota
	IgnoreBranch
	FollowBranch
	IncludeBranch
)

type Config struct{
	Selector Selector
	Limiter  Limiter
	Output   Output
	Target   pulse.Number
}

func (v Config) Ensure() {
	switch {
	case v.Limiter == nil:
		panic(throw.IllegalValue())
	case v.Output == nil:
		panic(throw.IllegalValue())
	case !v.Target.IsTimePulse():
		panic(throw.IllegalValue())
	}
}
