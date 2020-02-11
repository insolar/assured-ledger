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

import "github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"

var _ cryptkit.ForkingDigester = &ForkingCalculator{}

func NewForkingCalculator(digester cryptkit.PairDigester, unbalancedStub cryptkit.Digest) ForkingCalculator {
	if digester == nil {
		panic("illegal value")
	}
	return ForkingCalculator{NewStackedCalculator(digester, unbalancedStub, nil)}
}

type ForkingCalculator struct {
	StackedCalculator
}

func (p *ForkingCalculator) ForkSequence() cryptkit.ForkingDigester {
	if p.finished || p.digester == nil || p.traceFn != nil {
		panic("illegal state")
	}

	cp := *p
	cp.treeLevels = append(make([]treeLevel, 0, cap(p.treeLevels)), p.treeLevels...)
	return &cp
}
