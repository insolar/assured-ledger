// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package genesisrefs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenesisRef(t *testing.T) {
	t.SkipNow()
	var (
		pubKey    = "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEf+vsMVU75xH8uj5WRcOqYdHXtaHH\nN0na2RVQ1xbhsVybYPae3ujNHeQCPj+RaJyMVhb6Aj/AOsTTOPFswwIDAQ==\n-----END PUBLIC KEY-----\n"
		pubKeyRef = "insolar:1AAEAAcEp7HwQByGOr6rZwkyiRA3wR2POYCrDIhqBJyY"
	)
	genesisRef := GenesisRef(pubKey)
	require.Equal(t, pubKeyRef, genesisRef.String(), "reference by name always the same")
}
