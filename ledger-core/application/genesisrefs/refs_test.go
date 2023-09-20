package genesisrefs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenesisRef(t *testing.T) {
	var (
		pubKey    = "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEf+vsMVU75xH8uj5WRcOqYdHXtaHH\nN0na2RVQ1xbhsVybYPae3ujNHeQCPj+RaJyMVhb6Aj/AOsTTOPFswwIDAQ==\n-----END PUBLIC KEY-----\n"
		pubKeyRef = "insolar:1AAEAARI-eAuj-JtuPQVenwDG93tE6_zJzkA0hwRzSME"
	)
	genesisRef := GenesisRef(pubKey)
	require.Equal(t, pubKeyRef, genesisRef.String(), "reference by name always the same")
}
