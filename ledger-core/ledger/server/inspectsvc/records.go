// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package inspectsvc

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

type RegisterRequestSet = []*rms.LRegisterRequest

type InspectedRecordSet struct {

}

type InspectedRecord struct {
	Excerpt catalog.Excerpt
	Record  *rms.LRegisterRequest

	RegistrarSignature cryptkit.SignedDigest
}
