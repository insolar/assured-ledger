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
	"github.com/pkg/errors"

	XXX_insolar "github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	XXX_artifacts "github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/artifacts"
)

func InitializeContractMethods() map[string]XXX_insolar.ContractWrapper {
	return map[string]XXX_insolar.ContractWrapper{}
}

func shouldLoadRef(strRef string) XXX_insolar.Reference {
	ref, err := XXX_insolar.NewReferenceFromString(strRef)
	if err != nil {
		panic(errors.Wrap(err, "Unexpected error, bailing out"))
	}
	return *ref
}

func InitializeCodeRefs() map[XXX_insolar.Reference]string {
	rv := make(map[XXX_insolar.Reference]string, 0)

	return rv
}

func InitializePrototypeRefs() map[XXX_insolar.Reference]string {
	rv := make(map[XXX_insolar.Reference]string, 0)

	return rv
}

func InitializeCodeDescriptors() []XXX_artifacts.CodeDescriptor {
	rv := make([]XXX_artifacts.CodeDescriptor, 0, 0)

	return rv
}

func InitializePrototypeDescriptors() []XXX_artifacts.PrototypeDescriptor {
	rv := make([]XXX_artifacts.PrototypeDescriptor, 0, 0)

	return rv
}
