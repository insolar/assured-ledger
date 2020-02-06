//
//    Copyright 2019 Insolar Technologies
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.
//

package merkler

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/args"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
)

var _ cryptkit.ForkingDigester = &StackedCalculator{}

func NewStackedCalculator(digester cryptkit.PairDigester, unbalancedStub cryptkit.Digest) StackedCalculator {
	if digester == nil {
		panic("illegal value")
	}
	return StackedCalculator{digester: digester, unbalancedStub: unbalancedStub}
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
//  - FinishSequence() is O(log n), it does log(n) calls to PairDigester.DigestPair()
//  - ForkSequence() is O(log n), but only copies memory
//  - Memory is O(log n)
//
type StackedCalculator struct {
	digester       cryptkit.PairDigester
	unbalancedStub cryptkit.Digest
	prevAdded      longbits.FoldableReader
	count          uint
	treeLevels     []treeLevel
	finished       bool
	//sequenceFn     func (longbits.FoldableReader)
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

func (p *StackedCalculator) AddNext(addDigest longbits.FoldableReader) {

	if p.finished || p.digester == nil {
		panic("illegal state")
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
		bottomDigest = p.digester.DigestPair(p.prevAdded, addDigest)
		p.prevAdded = nil

		//if p.sequenceFn != nil {
		//	p.sequenceFn(addDigest)
		//	p.sequenceFn(bottomDigest)
		//}
	} else {
		if p.prevAdded != nil {
			panic("illegal state")
		}
		p.prevAdded = addDigest

		if int(pairPosition) > len(p.treeLevels) {
			return
		}
		pairLevel := &p.treeLevels[pairPosition-1]
		bottomDigest = p.digester.DigestPair(pairLevel.digest0, pairLevel.digest1)
		pairLevel.digest0, pairLevel.digest1 = cryptkit.Digest{}, cryptkit.Digest{}

		//if p.sequenceFn != nil {
		//	p.sequenceFn(bottomDigest)
		//	p.sequenceFn(addDigest)
		//}
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

func (p *StackedCalculator) ForkSequence() cryptkit.ForkingDigester {

	if p.finished || p.digester == nil {
		panic("illegal state")
	}

	cp := *p
	cp.treeLevels = append(make([]treeLevel, 0, cap(p.treeLevels)), p.treeLevels...)
	return &cp
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
			bottomDigest = p.digester.DigestPair(p.prevAdded, p.unbalancedStub)
		} else {
			bottomDigest = cryptkit.NewDigest(p.prevAdded, p.digester.GetDigestMethod())
		}
	}

	for i := 0; i < len(p.treeLevels); i++ { // DONT USE range as treeLevels can be appended inside the loop!
		curLevel := &p.treeLevels[i]

		switch {
		case !curLevel.digest1.IsEmpty():
			// both are present
			if curLevel.digest0.IsEmpty() {
				panic("illegal state")
			}
			levelDigest := p.digester.DigestPair(curLevel.digest0, curLevel.digest1)

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
					panic("illegal state - only one dual-hash can be present in the stack")
				}
			}
			if hasStub {
				bottomDigest = p.digester.DigestPair(bottomDigest, p.unbalancedStub)
			}
			// or leave as is
		case !curLevel.digest0.IsEmpty():
			switch {
			case !bottomDigest.IsEmpty():
				bottomDigest = p.digester.DigestPair(curLevel.digest0, bottomDigest)
			case hasStub:
				bottomDigest = p.digester.DigestPair(curLevel.digest0, p.unbalancedStub)
			default:
				bottomDigest = curLevel.digest0
			}
		case !bottomDigest.IsEmpty() && hasStub:
			bottomDigest = p.digester.DigestPair(bottomDigest, p.unbalancedStub)
		}
	}

	return bottomDigest
}
