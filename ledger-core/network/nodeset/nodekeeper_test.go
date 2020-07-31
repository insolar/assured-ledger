// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nodeset

import (
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/reference"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
)

func TestNewNodeNetwork(t *testing.T) {
	instestlogger.SetTestOutput(t)

	cfg := configuration.Transport{Address: "127.0.0.1:3355"}
	certMock := testutils.NewCertificateMock(t)
	certMock.GetRoleMock.Set(func() member.PrimaryRole { return member.PrimaryRoleLightMaterial })
	certMock.GetNodeRefMock.Set(func() reference.Global { ref := gen.UniqueGlobalRef(); return ref })

	_, err := NewNodeNetwork(cfg, certMock)
	assert.NoError(t, err)
}
