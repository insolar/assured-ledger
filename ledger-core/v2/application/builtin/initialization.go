//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

// Code generated by insgocc. DO NOT EDIT.
// source template in logicrunner/preprocessor/templates

package builtin

import (
	XXX_contract "github.com/insolar/assured-ledger/ledger-core/v2/insolar/contract"
	XXX_reference "github.com/insolar/assured-ledger/ledger-core/v2/reference"
	XXX_machine "github.com/insolar/assured-ledger/ledger-core/v2/runner/machine"
	throw "github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
	XXX_descriptor "github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"

	testwallet "github.com/insolar/assured-ledger/ledger-core/v2/application/builtin/contract/testwallet"
)

func InitializeContractMethods() map[string]XXX_contract.Wrapper {
	return map[string]XXX_contract.Wrapper{
		"testwallet": testwallet.Initialize(),
	}
}

func shouldLoadRef(strRef string) XXX_reference.Global {
	ref, err := XXX_reference.GlobalFromString(strRef)
	if err != nil {
		panic(throw.W(err, "Unexpected error, bailing out"))
	}
	return ref
}

func InitializeCodeRefs() map[XXX_reference.Global]string {
	rv := make(map[XXX_reference.Global]string, 1)

	rv[shouldLoadRef("insolar:0AAABAl_vPviVYDW1UkqOuygiJYr8FWd-7mDbJtjlwx4.record")] = "testwallet"

	return rv
}

func InitializeClassRefs() map[XXX_reference.Global]string {
	rv := make(map[XXX_reference.Global]string, 1)

	rv[shouldLoadRef("insolar:0AAABAqiF7kGalgYGa1bKDmA33RKr0lfmdtIZr73_tMU")] = "testwallet"

	return rv
}

func InitializeCodeDescriptors() []XXX_descriptor.Code {
	rv := make([]XXX_descriptor.Code, 0, 1)

	// testwallet
	rv = append(rv, XXX_descriptor.NewCode(
		/* code:        */ nil,
		/* machineType: */ XXX_machine.Builtin,
		/* ref:         */ shouldLoadRef("insolar:0AAABAl_vPviVYDW1UkqOuygiJYr8FWd-7mDbJtjlwx4.record"),
	))

	return rv
}

func InitializeClassDescriptors() []XXX_descriptor.Class {
	rv := make([]XXX_descriptor.Class, 0, 1)

	{ // testwallet
		pRef := shouldLoadRef("insolar:0AAABAqiF7kGalgYGa1bKDmA33RKr0lfmdtIZr73_tMU")
		cRef := shouldLoadRef("insolar:0AAABAl_vPviVYDW1UkqOuygiJYr8FWd-7mDbJtjlwx4.record")
		rv = append(rv, XXX_descriptor.NewClass(
			/* head:         */ pRef,
			/* state:        */ pRef.GetLocal(),
			/* code:         */ cRef,
		))
	}

	return rv
}
