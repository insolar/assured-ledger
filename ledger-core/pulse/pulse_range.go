// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulse

import "math"

type FindNumberFunc func(n Number, prevDelta, nextDelta uint16) bool

type Range interface {
	// LeftBoundNumber provides a left bound of the range. It may be an expected pulse.
	LeftBoundNumber() Number
	// LeftPrevDelta is PrevDelta associated with the left boundary (refers to a pulse _before_ the left boundary).
	LeftPrevDelta() uint16

	// RightBoundData returns a right bound of the range. MUST be a valid or expected pulse data.
	RightBoundData() Data

	// IsArticulated indicates that this range requires articulated pulses to be properly chained.
	IsArticulated() bool
	// IsSingular indicates that this range is a singular and contains only one pulse.
	IsSingular() bool

	// EnumNumbers iterates from smaller to higher pulses, over both provided and articulated pulses within the range.
	EnumNumbers(fn FindNumberFunc) bool
	// EnumNonArticulatedNumbers iterates from smaller to higher pulses, only over the provided pulse data within the range.
	EnumNonArticulatedNumbers(fn FindNumberFunc) bool

	// EnumData iterates from smaller to higher pulses, over both provided and articulated pulse data within the range.
	EnumData(func(Data) bool) bool
	// EnumNonArticulatedData iterates from smaller to higher pulses, only over the provided pulse data within the range.
	EnumNonArticulatedData(func(Data) bool) bool

	// IsValidNext returns true then the given range is next immediate range
	IsValidNext(Range) bool
	// IsValidPrev returns true then the given range is prev immediate range
	IsValidPrev(Range) bool
	// Equal returns true when both ranges are equal
	Equal(Range) bool
}

// Creates a range that covers a gap between the last expected pulse and the last available one.
// Will panic when pulse are overlapping and can't be properly connected.
// Supports gaps with delta > 65535
func NewLeftGapRange(left Number, leftPrevDelta uint16, right Data) Range {
	right.EnsurePulseData()
	switch {
	case left == right.PulseNumber:
		if leftPrevDelta == right.PrevPulseDelta {
			return right.AsRange()
		}
	case left.IsBeforeOrEq(right.PrevPulseNumber()):
		left.Prev(leftPrevDelta) // ensure correctness
		return gapPulseRange{start: left, prevDelta: leftPrevDelta, end: right}
	}
	panic("illegal value")
}

// Creates a range that consists of >0 properly connected pulses.
// Sequence MUST be sorted, all pulses must be connected, otherwise use NewPulseRange()
// Will panic when pulse are overlapping and can't be properly connected.
// Supports gaps with delta > 65535
func NewSequenceRange(data []Data) Range {
	switch {
	case len(data) == 0:
		panic("illegal value")
	case len(data) == 1:
		return data[0].AsRange()
	case checkSequence(data):
		return seqPulseRange{data: append([]Data(nil), data...)}
	}
	panic("illegal value")
}

// Creates a range that consists of both connected and disconnected pulses.
// Sequence MUST be sorted, an expected pulse is allowed at [0]
// Will panic when pulse are overlapping and can't be properly connected.
// Supports gaps with delta > 65535
func NewPulseRange(data []Data) Range {
	switch {
	case len(data) == 0:
		panic("illegal value")
	case len(data) == 1:
		return data[0].AsRange()
	case !data[0].isExpected():
		if checkSequence(data) {
			return seqPulseRange{data: append([]Data(nil), data...)}
		}
	case data[1].PrevPulseNumber() < data[0].PulseNumber || !data[0].IsValidExpectedPulseData():
		panic("illegal value")
	case len(data) == 2:
		return NewLeftGapRange(data[0].PulseNumber, data[0].PrevPulseDelta, data[1])
	default:
		checkSequence(data[1:])
	}
	return sparsePulseRange{data: append([]Data(nil), data...)}
}

func checkSequence(data []Data) bool {
	sequence := true
	for i, d := range data {
		d.EnsurePulseData()
		if i == 0 {
			continue
		}

		prev := &data[i-1]
		switch {
		case prev.IsValidNext(d):
		case prev.NextPulseNumber() > d.PrevPulseNumber():
			panic("illegal value - unordered or intersecting pulses")
		default:
			sequence = false
		}
	}
	return sequence
}

/* ===================================================== */

var _ Range = OnePulseRange{}

func NewOnePulseRange(data Data) OnePulseRange {
	data.ensureRangeData()
	return OnePulseRange{data}
}

type OnePulseRange struct {
	data Data
}

func (p OnePulseRange) IsValidNext(a Range) bool {
	if p.data.NextPulseDelta == 0 || p.data.NextPulseDelta != a.LeftPrevDelta() {
		return false
	}
	if n, ok := p.data.GetNextPulseNumber(); ok {
		return n == a.LeftBoundNumber()
	}
	return false
}

func (p OnePulseRange) IsValidPrev(a Range) bool {
	return a.IsValidNext(p)
}

func (p OnePulseRange) EnumNumbers(fn FindNumberFunc) bool {
	return fn(p.data.PulseNumber, p.data.PrevPulseDelta, p.data.NextPulseDelta)
}

func (p OnePulseRange) EnumData(fn func(Data) bool) bool {
	return fn(p.data)
}

func (p OnePulseRange) EnumNonArticulatedNumbers(fn FindNumberFunc) bool {
	return fn(p.data.PulseNumber, p.data.PrevPulseDelta, p.data.NextPulseDelta)
}

func (p OnePulseRange) EnumNonArticulatedData(fn func(Data) bool) bool {
	return fn(p.data)
}

func (p OnePulseRange) RightBoundData() Data {
	return p.data
}

func (p OnePulseRange) IsSingular() bool {
	return true
}

func (p OnePulseRange) IsArticulated() bool {
	return false
}

func (p OnePulseRange) LeftBoundNumber() Number {
	return p.data.PulseNumber
}

func (p OnePulseRange) LeftPrevDelta() uint16 {
	return p.data.PrevPulseDelta
}

func (p OnePulseRange) Equal(o Range) bool {
	if po, ok := o.(OnePulseRange); ok {
		return p == po
	}
	return false
}

/* ===================================================== */

type templatePulseRange struct {
}

func (p templatePulseRange) IsSingular() bool {
	return false
}

var _ Range = gapPulseRange{}

type gapPulseRange struct {
	templatePulseRange
	start     Number
	prevDelta uint16
	end       Data
}

func (p gapPulseRange) IsValidNext(a Range) bool {
	if p.end.NextPulseDelta != a.LeftPrevDelta() {
		return false
	}
	if n, ok := p.end.GetNextPulseNumber(); ok {
		return n == a.LeftBoundNumber()
	}
	return false
}

func (p gapPulseRange) IsValidPrev(a Range) bool {
	return a.IsValidNext(p)
}

func (p gapPulseRange) EnumNumbers(fn FindNumberFunc) bool {
	if _enumSegments(p.start, p.prevDelta, p.end.PrevPulseNumber(), p.end.PrevPulseDelta, fn) {
		return true
	}
	return fn(p.end.PulseNumber, p.end.PrevPulseDelta, p.end.NextPulseDelta)
}

func (p gapPulseRange) EnumNonArticulatedNumbers(fn FindNumberFunc) bool {
	return fn(p.end.PulseNumber, p.end.PrevPulseDelta, p.end.NextPulseDelta)
}

func (p gapPulseRange) EnumData(fn func(Data) bool) bool {
	return _enumSegmentData(p.start, p.prevDelta, p.end, fn)
}

func (p gapPulseRange) EnumNonArticulatedData(fn func(Data) bool) bool {
	return fn(p.end)
}

func (p gapPulseRange) LeftPrevDelta() uint16 {
	return p.prevDelta
}

func (p gapPulseRange) LeftBoundNumber() Number {
	return p.start
}

func (p gapPulseRange) RightBoundData() Data {
	return p.end
}

func (p gapPulseRange) IsArticulated() bool {
	return true
}

func (p gapPulseRange) Equal(o Range) bool {
	if po, ok := o.(gapPulseRange); ok {
		return p == po
	}
	return false
}

/* ===================================================== */
var _ Range = seqPulseRange{}

type seqPulseRange struct {
	templatePulseRange
	data []Data
}

func (p seqPulseRange) Equal(o Range) bool {
	if po, ok := o.(seqPulseRange); ok {
		if len(p.data) != len(po.data) {
			return false
		}
		for i := range p.data {
			if p.data[i] != po.data[i] {
				return false
			}
		}
		return true
	}
	return false
}

func (p seqPulseRange) IsValidNext(a Range) bool {
	end := p.RightBoundData()
	if end.NextPulseDelta != a.LeftPrevDelta() {
		return false
	}
	if n, ok := end.GetNextPulseNumber(); ok {
		return n == a.LeftBoundNumber()
	}
	return false
}

func (p seqPulseRange) IsValidPrev(a Range) bool {
	return a.IsValidNext(p)
}

func (p seqPulseRange) EnumNumbers(fn FindNumberFunc) bool {
	return p.EnumNonArticulatedNumbers(fn)
}

func (p seqPulseRange) EnumNonArticulatedNumbers(fn FindNumberFunc) bool {
	if fn == nil {
		panic("illegal value")
	}
	for _, d := range p.data {
		if fn(d.PulseNumber, d.PrevPulseDelta, d.NextPulseDelta) {
			return true
		}
	}
	return false
}

func (p seqPulseRange) EnumData(fn func(Data) bool) bool {
	return p.EnumNonArticulatedData(fn)
}

func (p seqPulseRange) EnumNonArticulatedData(fn func(Data) bool) bool {
	if fn == nil {
		panic("illegal value")
	}
	for _, d := range p.data {
		if fn(d) {
			return true
		}
	}
	return false
}

func (p seqPulseRange) IsArticulated() bool {
	return false
}

func (p seqPulseRange) LeftPrevDelta() uint16 {
	return p.data[0].PrevPulseDelta
}

func (p seqPulseRange) LeftBoundNumber() Number {
	return p.data[0].PulseNumber
}

func (p seqPulseRange) RightBoundData() Data {
	return p.data[len(p.data)-1]
}

/* ===================================================== */
var _ Range = sparsePulseRange{}

type sparsePulseRange struct {
	templatePulseRange
	data []Data
}

func (p sparsePulseRange) Equal(o Range) bool {
	if po, ok := o.(sparsePulseRange); ok {
		if len(p.data) != len(po.data) {
			return false
		}
		for i := range p.data {
			if p.data[i] != po.data[i] {
				return false
			}
		}
		return true
	}
	return false
}

func (p sparsePulseRange) IsValidNext(a Range) bool {
	end := p.RightBoundData()
	if end.NextPulseDelta != a.LeftPrevDelta() {
		return false
	}
	if n, ok := end.GetNextPulseNumber(); ok {
		return n == a.LeftBoundNumber()
	}
	return false
}

func (p sparsePulseRange) IsValidPrev(a Range) bool {
	return a.IsValidNext(p)
}

func (p sparsePulseRange) EnumNumbers(fn FindNumberFunc) bool {
	var (
		next      Number
		prevDelta uint16
	)

	if fn == nil {
		panic("illegal value")
	}

	switch first := p.data[0]; {
	case first.NextPulseDelta == 0: // expected pulse
		next, prevDelta = first.PulseNumber, first.PrevPulseDelta
	default:
		next, prevDelta = first.NextPulseNumber(), first.NextPulseDelta
		fn(first.PulseNumber, first.PrevPulseDelta, first.NextPulseDelta)
	}

	for _, d := range p.data[1:] {
		switch {
		case next == d.PulseNumber:
			if _enumSegments(next, prevDelta, d.PulseNumber, d.NextPulseDelta, fn) {
				return true
			}
		case _enumSegments(next, prevDelta, d.PrevPulseNumber(), d.PrevPulseDelta, fn):
			return true
		case fn(d.PulseNumber, d.PrevPulseDelta, d.NextPulseDelta):
			return true
		}
		next, prevDelta = d.NextPulseNumber(), d.NextPulseDelta
	}
	return false
}

func (p sparsePulseRange) EnumData(fn func(Data) bool) bool {
	var (
		next      Number
		prevDelta uint16
	)

	if fn == nil {
		panic("illegal value")
	}

	switch first := p.data[0]; {
	case first.NextPulseDelta == 0: // expected pulse
		next, prevDelta = first.PulseNumber, first.PrevPulseDelta
	default:
		next, prevDelta = first.NextPulseNumber(), first.NextPulseDelta
		fn(first)
	}

	for _, d := range p.data[1:] {
		if _enumSegmentData(next, prevDelta, d, fn) {
			return true
		}
		next, prevDelta = d.NextPulseNumber(), d.NextPulseDelta
	}
	return false
}

func (p sparsePulseRange) EnumNonArticulatedNumbers(fn FindNumberFunc) bool {
	if fn == nil {
		panic("illegal value")
	}

	startIdx := 0
	if p.data[0].NextPulseDelta == 0 {
		startIdx++
	}

	for _, d := range p.data[startIdx:] {
		if fn(d.PulseNumber, d.PrevPulseDelta, d.NextPulseDelta) {
			return true
		}
	}
	return false
}

func (p sparsePulseRange) EnumNonArticulatedData(fn func(Data) bool) bool {
	if fn == nil {
		panic("illegal value")
	}

	startIdx := 0
	if p.data[0].NextPulseDelta == 0 {
		startIdx++
	}

	for _, d := range p.data[startIdx:] {
		if fn(d) {
			return true
		}
	}
	return false
}

func (p sparsePulseRange) IsArticulated() bool {
	return true
}

func (p sparsePulseRange) LeftPrevDelta() uint16 {
	return p.data[0].PrevPulseDelta
}

func (p sparsePulseRange) LeftBoundNumber() Number {
	return p.data[0].PulseNumber
}

func (p sparsePulseRange) RightBoundData() Data {
	return p.data[len(p.data)-1]
}

/* ===================================================== */
const minSegmentPulseDelta = 10

func _enumSegmentData(start Number, prevDelta uint16, end Data, fn func(Data) bool) bool {
	if start != end.PulseNumber && _enumSegments(start, prevDelta, end.PrevPulseNumber(), end.PrevPulseDelta,
		func(n Number, prevDelta, nextDelta uint16) bool {
			return fn(Data{
				n, DataExt{
					PulseEpoch:     ArticulationPulseEpoch,
					NextPulseDelta: nextDelta,
					PrevPulseDelta: prevDelta,
				}})
		}) {
		return true
	}
	return fn(end)
}

func _enumSegments(next Number, prevDelta uint16, end Number, endNextDelta uint16, fn FindNumberFunc) bool {
	for {
		switch {
		case next < end:
			delta := end - next
			switch {
			case delta <= math.MaxUint16:
			case delta < math.MaxUint16+minSegmentPulseDelta:
				delta -= minSegmentPulseDelta
			default:
				delta = math.MaxUint16
			}
			if fn(next, prevDelta, uint16(delta)) {
				return true
			}
			prevDelta = uint16(delta)
			next = next.Next(prevDelta)
			continue
		case next == end:
			return fn(next, prevDelta, endNextDelta)
		default:
			panic("illegal state")
		}
	}
}
