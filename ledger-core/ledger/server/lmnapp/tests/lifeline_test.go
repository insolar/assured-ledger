// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package tests

import (
	"context"
	"strconv"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/inspectsvc"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lmnapp/lmntestapp"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

type generatorNewLifeline struct {
	recBuilder lmntestapp.RecordBuilder
	seqNo      atomickit.Uint32
	totalBytes atomickit.Uint64
	body       []byte
	conv       *conveyor.PulseConveyor
}

func (p *generatorNewLifeline) makeSet(reasonRef reference.Holder) inspectsvc.RegisterRequestSet {

	rb, rootRec := p.recBuilder.MakeLineStart(&rms.RLifelineStart{
		Str: strconv.Itoa(int(p.seqNo.Add(1))),
	})
	rootRec.OverrideRecordType = rms.TypeRLifelineStartPolymorphID
	rootRec.OverrideReasonRef.Set(reasonRef)

	rMem := &rms.RLineMemoryInit{
		Polymorph: rms.TypeRLineMemoryInitPolymorphID,
		RootRef:   rootRec.AnticipatedRef,
		PrevRef:   rootRec.AnticipatedRef,
	}
	rMem.SetDigester(rb.RecordScheme.RecordDigester())
	rMem.SetPayload(rms.NewRawBytes(p.body))

	rq := rb.Add(rMem)

	rq = rb.Add(&rms.RLineActivate{
		RootRef: rootRec.AnticipatedRef,
		PrevRef: rq.AnticipatedRef,
	})

	return rb.MakeSet()
}

func (p *generatorNewLifeline) sumUpRegister(recordSet inspectsvc.RegisterRequestSet) {
	setSize := 0
	for _, r := range recordSet.Requests {
		setSize += r.ProtoSize()
		rp := r.GetRecordPayloads()
		setSize += rp.ProtoSize()
	}

	p.totalBytes.Add(uint64(setSize))
}

func (p *generatorNewLifeline) callRegister(recordSet inspectsvc.RegisterRequestSet) ([]cryptkit.Signature, error) {
	pn := p.recBuilder.RefTemplate.LocalHeader().Pulse()
	return p._call(pn, recordSet)
}

func (p *generatorNewLifeline) callVerify(regRecordSet inspectsvc.RegisterRequestSet) ([]cryptkit.Signature, error) {
	pn := p.recBuilder.RefTemplate.LocalHeader().Pulse()

	var recordSet inspectsvc.VerifyRequestSet
	recordSet.Excerpt = regRecordSet.Excerpt
	recordSet.Requests = make([]*rms.LRegisterRequest, len(regRecordSet.Requests))

	for i, r := range regRecordSet.Requests {
		rc := *r
		rc.AnyRecordLazy = r.AnyRecordLazy.CopyNoPayloads()
		recordSet.Requests[i] = &rc
	}

	return p._call(pn, recordSet)
}

func (p *generatorNewLifeline) _call(pn pulse.Number, recordSet interface{}) ([]cryptkit.Signature, error) {
	ch := make(chan smachine.TerminationData, 1)
	err := p.conv.AddInputExt(pn,
		recordSet,
		smachine.CreateDefaultValues{
			Context: context.Background(),
			TerminationHandler: func(data smachine.TerminationData) {
				ch <- data
				close(ch)
			},
		})
	if err != nil {
		panic(err)
	}
	data := <-ch
	if data.Result == nil {
		return nil, data.Error
	}

	return data.Result.([]cryptkit.Signature), data.Error
}

func (p *generatorNewLifeline) registerNewLine(reasonRef reference.Holder) ([]cryptkit.Signature, error) {
	recordSet := p.makeSet(reasonRef)
	p.sumUpRegister(recordSet)
	return p.callRegister(recordSet)
}

func (p *generatorNewLifeline) callRead(ref reference.Holder) (*rms.LReadResponse, error) {
	pn := p.recBuilder.RefTemplate.LocalHeader().Pulse()

	request := &rms.LReadRequest{}
	request.TargetRef.Set(ref)

	ch := make(chan smachine.TerminationData, 1)
	err := p.conv.AddInputExt(pn,
		request,
		smachine.CreateDefaultValues{
			Context: context.Background(),
			TerminationHandler: func(data smachine.TerminationData) {
				ch <- data
				close(ch)
			},
		})
	if err != nil {
		panic(err)
	}
	data := <-ch
	if data.Result == nil {
		return nil, data.Error
	}

	return data.Result.(*rms.LReadResponse), data.Error
}

func (p *generatorNewLifeline) sumUpRead(response *rms.LReadResponse) {
	setSize := 0
	for _, r := range response.Entries {
		setSize += r.ProtoSize()
	}

	p.totalBytes.Add(uint64(setSize))
}

