// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build tools

package tools

import (
	_ "github.com/cyraxred/go-acc"
	_ "github.com/gogo/protobuf/protoc-gen-gogofaster"
	_ "github.com/gogo/protobuf/protoc-gen-gogoslick"
	_ "github.com/gojuno/minimock/v3/cmd/minimock"
	_ "github.com/golang/protobuf/protoc-gen-go"
	_ "github.com/insolar/sm-uml-gen"
	_ "golang.org/x/tools/cmd/goimports"
	_ "golang.org/x/tools/cmd/stringer"

	_ "github.com/insolar/assured-ledger/ledger-core/virtual/integration/mock/publisher/cmd/insolar-mockgenerator-typedpublisher"
	_ "github.com/insolar/assured-ledger/ledger-core/virtual/integration/mock/referencebuilder/cmd/insolar-mockgenerator-referencebuilder"
)
