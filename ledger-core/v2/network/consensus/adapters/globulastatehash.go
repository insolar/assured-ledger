// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package adapters

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/gcpv2/api/transport"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
)

func NewSequenceDigester(dataDigester cryptkit.DataDigester) *SequenceDigester {
	return &SequenceDigester{
		dataDigester: dataDigester,
	}
}

type SequenceDigester struct {
	dataDigester cryptkit.DataDigester
	state        uint64
}

func (d *SequenceDigester) GetDigestSize() int {
	return d.dataDigester.GetDigestSize()
}

func (d *SequenceDigester) AddNext(digest longbits.FoldableReader) {
	d.addNext(digest.FoldToUint64())
}

func (d *SequenceDigester) addNext(state uint64) {
	d.state ^= state
}

func (d *SequenceDigester) FinishSequence() cryptkit.Digest {
	bits64 := longbits.NewBits64(d.state)
	return d.dataDigester.DigestData(&bits64)
}

func (d *SequenceDigester) GetDigestMethod() cryptkit.DigestMethod {
	return d.dataDigester.GetDigestMethod()
}

func (d *SequenceDigester) ForkSequence() cryptkit.ForkingDigester {
	return &SequenceDigester{
		dataDigester: d.dataDigester,
		state:        d.state,
	}
}

type StateDigester struct {
	sequenceDigester *SequenceDigester
	defaultDigest    longbits.FoldableReader
}

func NewStateDigester(sequenceDigester *SequenceDigester) *StateDigester {
	return &StateDigester{
		sequenceDigester: sequenceDigester,
		defaultDigest:    &longbits.Bits512{},
	}
}

func (d *StateDigester) AddNext(digest longbits.FoldableReader, fullRank member.FullRank) {
	if digest == nil {
		d.sequenceDigester.AddNext(d.defaultDigest)
	} else {
		d.sequenceDigester.AddNext(digest)
		d.sequenceDigester.addNext(uint64(fullRank.AsMembershipRank(member.MaxNodeIndex)))
	}
}

func (d *StateDigester) GetDigestMethod() cryptkit.DigestMethod {
	return d.sequenceDigester.GetDigestMethod()
}

func (d *StateDigester) ForkSequence() transport.StateDigester {
	return &StateDigester{
		sequenceDigester: d.sequenceDigester.ForkSequence().(*SequenceDigester),
		defaultDigest:    d.defaultDigest,
	}
}

func (d *StateDigester) FinishSequence() cryptkit.Digest {
	return d.sequenceDigester.FinishSequence()
}
