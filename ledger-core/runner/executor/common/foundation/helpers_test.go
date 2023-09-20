package foundation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrimPublicKey(t *testing.T) {
	for _, tc := range []struct {
		input  string
		result string
	}{
		{
			input:  "asDafasf",
			result: "asDafasf",
		},
		{
			input:  "-----BEGIN RSA PUBLIC KEY-----\naSDafasf\n-----END RSA PUBLIC KEY-----",
			result: "aSDafasf",
		},
	} {
		require.Equal(t, tc.result, TrimPublicKey(tc.input))
	}
}
