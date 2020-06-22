// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package object

import (
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type ResultTable struct {
	lists map[contract.InterferenceFlag]*ResultList
}

func (rt *ResultTable) Len() int {
	size := 0
	for _, list := range rt.lists {
		size += list.Count()
	}
	return size
}

func NewResultTable() ResultTable {
	var rt ResultTable
	rt.lists = make(map[contract.InterferenceFlag]*ResultList)

	rt.lists[contract.CallTolerable] = NewResultList()
	rt.lists[contract.CallIntolerable] = NewResultList()
	return rt
}

func (rt *ResultTable) GetList(flag contract.InterferenceFlag) *ResultList {
	if flag.IsZero() {
		panic(throw.IllegalValue())
	}
	return rt.lists[flag]
}

type ResultList struct {
	results map[reference.Global]*payload.VCallResult
}

func NewResultList() *ResultList {
	return &ResultList{
		results: make(map[reference.Global]*payload.VCallResult),
	}
}

func (rl *ResultList) Add(ref reference.Global, result *payload.VCallResult) bool {
	if _, exist := rl.results[ref]; exist {
		return false
	}

	rl.results[ref] = result

	return true
}

func (rl *ResultList) Get(ref reference.Global) (*payload.VCallResult, bool) {
	result, ok := rl.results[ref]
	return result, ok
}

func (rl *ResultList) Count() int {
	return len(rl.results)
}
