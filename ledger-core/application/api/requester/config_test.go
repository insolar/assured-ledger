package requester

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
)

func TestReadFile_BadFile(t *testing.T) {
	instestlogger.SetTestOutput(t)

	err := readFile("zzz", nil)
	require.Contains(t, err.Error(), "[ readFile ] Problem with reading config;\topen zzz:")
}

func TestReadFile_NotJson(t *testing.T) {
	instestlogger.SetTestOutput(t)

	err := readFile("testdata/bad_json.json", nil)
	require.EqualError(t, err, "[ readFile ] Problem with unmarshaling config;\tinvalid character ']' after object key")
}

func TestReadRequestConfigFromFile(t *testing.T) {
	instestlogger.SetTestOutput(t)

	params, err := ReadRequestParamsFromFile("testdata/requestConfig.json")
	require.NoError(t, err)

	require.Equal(t, "member.create", params.CallSite)
}

func TestReadUserConfigFromFile(t *testing.T) {
	instestlogger.SetTestOutput(t)

	conf, err := ReadUserConfigFromFile("testdata/userConfig.json")
	require.NoError(t, err)
	require.Contains(t, conf.PrivateKey, "MHcCAQEEIPOsF3ujjM7jnb7V")
	require.Equal(t, "4FFB8zfQoGznSmzDxwv4njX1aR9ioL8GHSH17QXH2AFa.4K3NiGuqYGqKPnYp6XeGd2kdN4P9veL6rYcWkLKWXZCu", conf.Caller)
}
