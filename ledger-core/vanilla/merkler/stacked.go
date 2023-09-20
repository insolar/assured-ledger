package merkler

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/args"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

var _ cryptkit.SequenceDigester = &StackedCalculator{}

type TraceFunc func(v longbits.FoldableReader, isLeaf bool)

// Creates a sequence digester that does merkle tree calculation.
// When (unbalancedStub) is empty, then the resulting merkle-tree will be closed without stubbing, otherwise with per-level stubbing with (unbalancedStub) on the right side.
// When (traceFn) is not nil, then StackedCalculator will call it each time when a leaf or node is added.
//
// NB! For every AddNext() call the (traceFn) is always called twice - first time with (leafX, true) value, and then (nodeX, false), where nodeX can be nil.
// And during FinishSequence() - the (traceFn) can be called N < 2*log(leafCount) times with (nodeX, false), where nodeX can not be nil.
//
func NewStackedCalculator(digester cryptkit.PairDigester, unbalancedStub cryptkit.Digest, traceFn TraceFunc) StackedCalculator {
	if digester == nil {
		panic("illegal value")
	}
	return StackedCalculator{digester: digester, unbalancedStub: unbalancedStub, traceFn: traceFn}
}

// A calculator to do streaming calculation of Merkle-tree by using provided PairDigester.
// The resulting merkle will have same depth for all branches except for the rightmost branch.
//
// When unbalancedStub == nil, then FinishSequence() will create the rightmost branch by recursively
// applying the same rule - all sub-branches will have same depth except for the rightmost sub-branch.
//
// When unbalancedStub != nil, then FinishSequence() will create a perfect tree by using unbalancedStub
// once per level when a value for the rightmost sub-branch is missing.
//
// When AddNext() was never called then FinishSequence() will return a non-nil unbalancedStub otherwise will panic.
//
// Complexity (n - a number of added hashes):
//  - AddNext() is O(1), it does only upto 2 calls to PairDigester.DigestPair()
//  - FinishSequence() is O(log n), it does k*log(n) calls to PairDigester.DigestPair() where k ~ 1 when unbalancedStub == nil and k ~ 2 otherwise
//  - ForkSequence() is O(log n), but only copies memory
//  - Memory is O(log n)
//
type StackedCalculator struct {
	digester       cryptkit.PairDigester
	unbalancedStub cryptkit.Digest
	prevAdded      longbits.FoldableReader
	traceFn        func(v longbits.FoldableReader, isLeaf bool)
	count          uint
	treeLevels     []treeLevel
	finished       bool
}

type treeLevel struct {
	digest0 cryptkit.Digest
	digest1 cryptkit.Digest
}

func (p *StackedCalculator) GetDigestMethod() cryptkit.DigestMethod {
	return p.digester.GetDigestMethod() + "/merkle"
}

func (p *StackedCalculator) GetDigestSize() int {
	return p.digester.GetDigestSize()
}

func (p *StackedCalculator) digest(v0, v1 longbits.FoldableReader) cryptkit.Digest {
	d := p.digester.DigestPair(v0, v1)
	if p.traceFn != nil {
		p.traceFn(d, false)
	}
	return d
}

func (p *StackedCalculator) AddNext(addDigest longbits.FoldableReader) {

	if p.finished || p.digester == nil {
		panic("illegal state")
	}

	if p.traceFn != nil {
		p.traceFn(addDigest, true)
	}

	/*
		Here we use position of grey-code transition bit to choose the level that requires pair-hashing
		Tree level is counted from a leaf to the root, leaf level is considered as -1
	*/

	p.count++
	pairPosition := args.GreyIncBit(p.count) // level + 1

	var bottomDigest cryptkit.Digest

	if pairPosition == 0 {
		// Level -1 (leafs) is special - it only stores a previous value
		bottomDigest = p.digest(p.prevAdded, addDigest)
		p.prevAdded = nil
	} else {
		if p.prevAdded != nil {
			panic("illegal state")
		}
		p.prevAdded = addDigest

		if int(pairPosition) > len(p.treeLevels) {
			if p.traceFn != nil {
				p.traceFn(nil, false)
			}
			return
		}
		pairLevel := &p.treeLevels[pairPosition-1]
		bottomDigest = p.digest(pairLevel.digest0, pairLevel.digest1)
		pairLevel.digest0, pairLevel.digest1 = cryptkit.Digest{}, cryptkit.Digest{}
	}

	if int(pairPosition) == len(p.treeLevels) {
		p.treeLevels = append(p.treeLevels, treeLevel{digest0: bottomDigest})
		return
	}

	var d *cryptkit.Digest
	if p.count&(uint(2)<<pairPosition) != 0 {
		d = &p.treeLevels[pairPosition].digest0
	} else {
		d = &p.treeLevels[pairPosition].digest1
	}
	if !d.IsEmpty() {
		panic("illegal state")
	}
	*d = bottomDigest
}

func (p *StackedCalculator) Count() int {
	return int(p.count)
}

func (p *StackedCalculator) FinishSequence() cryptkit.Digest {

	if p.finished || p.digester == nil {
		panic("illegal state")
	}
	p.finished = true

	hasStub := !p.unbalancedStub.IsEmpty()
	if p.count == 0 {
		if hasStub {
			return p.unbalancedStub
		}
		panic("illegal state - empty")
	}

	var bottomDigest cryptkit.Digest
	if p.prevAdded != nil {
		if hasStub {
			bottomDigest = p.digest(p.prevAdded, p.unbalancedStub)
		} else {
			bottomDigest = cryptkit.NewDigest(p.prevAdded, p.digester.GetDigestMethod())
		}
	}

	// TODO refactor/simplify
	for i := 0; i < len(p.treeLevels); i++ { // DONT USE range as treeLevels can be appended inside the loop!
		curLevel := &p.treeLevels[i]

		switch {
		case !curLevel.digest1.IsEmpty():
			// both are present
			if curLevel.digest0.IsEmpty() {
				panic("illegal state")
			}
			levelDigest := p.digest(curLevel.digest0, curLevel.digest1)

			if bottomDigest.IsEmpty() {
				bottomDigest = levelDigest
				continue
			}

			if i+1 == len(p.treeLevels) {
				p.treeLevels = append(p.treeLevels, treeLevel{digest0: levelDigest})
			} else {
				nextLevel := &p.treeLevels[i+1]
				switch {
				case nextLevel.digest0.IsEmpty():
					if !nextLevel.digest1.IsEmpty() {
						panic("illegal state")
					}
					nextLevel.digest0 = levelDigest
				case nextLevel.digest1.IsEmpty():
					nextLevel.digest1 = levelDigest
				default:
					panic("illegal state")
				}
			}
			if hasStub {
				bottomDigest = p.digest(bottomDigest, p.unbalancedStub)
			}
			// or leave as is
		case !curLevel.digest0.IsEmpty():
			switch {
			case !bottomDigest.IsEmpty():
				bottomDigest = p.digest(curLevel.digest0, bottomDigest)
			case hasStub && i < len(p.treeLevels)-1: // never stub the last level
				bottomDigest = p.digest(curLevel.digest0, p.unbalancedStub)
			default:
				bottomDigest = curLevel.digest0
			}
		case !bottomDigest.IsEmpty() && hasStub:
			bottomDigest = p.digest(bottomDigest, p.unbalancedStub)
		}
	}

	// return p.treeLevels[len(p.treeLevels)-1].digest0
	return bottomDigest
}
