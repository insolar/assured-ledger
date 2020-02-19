// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package genesisrefs

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/platformpolicy"
)

var (
	idHex  = "00010001c896b5c98f56001c688bc80a48274ac266a780a5a7ae74c4e1e3624b"
	refHex = idHex + idHex
)

func TestID(t *testing.T) {
	rootRecord := &Record{
		PCS: initPCS(),
	}
	require.Equal(t, idHex, hex.EncodeToString(rootRecord.ID().Bytes()), "root domain ID should always be the same")
}

func TestReference(t *testing.T) {
	rootRecord := &Record{
		PCS: initPCS(),
	}
	require.Equal(t, refHex, hex.EncodeToString(rootRecord.Reference().Bytes()), "root domain Ref should always be the same")

}

func initPCS() insolar.PlatformCryptographyScheme {
	return platformpolicy.NewPlatformCryptographyScheme()
}
