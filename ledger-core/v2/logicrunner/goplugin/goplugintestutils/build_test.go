// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package goplugintestutils

var contractOneCode = `
package main

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/builtin/foundation"
	recursive "github.com/insolar/assured-ledger/ledger-core/v2/application/proxy/recursive_call_one"
)
type One struct {
	foundation.BaseContract
}

func New() (*One, error) {
	return &One{}, nil
}

var INSATTR_Recursive_API = true
func (r *One) Recursive() (error) {
	remoteSelf := recursive.GetObject(r.GetReference())
	err := remoteSelf.Recursive()
	return err
}

`
