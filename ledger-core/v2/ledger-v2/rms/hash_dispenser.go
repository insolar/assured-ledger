// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type HashDispenser struct {
	state atomickit.OnceFlag
	value cryptkit.Digest
}

func (v *HashDispenser) Get() cryptkit.Digest {
	if v.IsSet() {
		return v.value
	}
	return cryptkit.Digest{}
}

func (v *HashDispenser) IsSet() bool {
	return v != nil && v.state.IsSet()
}

func (v *HashDispenser) set(value cryptkit.Digest) {
	if !v.state.DoSet(func() {
		v.value = value
	}) {
		panic(throw.IllegalState())
	}
}
