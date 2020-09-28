// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniproto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPLikeness(t *testing.T) {
	h := Header{}
	require.Equal(t, ErrPossibleHTTPRequest, h.DeserializeMinFromBytes([]byte("GET /0123456789ABCDEF")))

	require.Equal(t, ErrPossibleHTTPRequest, h.DeserializeMinFromBytes([]byte("PUT /0123456789ABCDEF")))

	require.Equal(t, ErrPossibleHTTPRequest, h.DeserializeMinFromBytes([]byte("POST /0123456789ABCDEF")))
	require.Equal(t, ErrPossibleHTTPRequest, h.DeserializeMinFromBytes([]byte("POST 0123456789ABCDEF")))

	require.Equal(t, ErrPossibleHTTPRequest, h.DeserializeMinFromBytes([]byte("HEAD /0123456789ABCDEF")))
	require.Equal(t, ErrPossibleHTTPRequest, h.DeserializeMinFromBytes([]byte("HEAD 0123456789ABCDEF")))

	require.Equal(t, ErrPossibleHTTPRequest, h.DeserializeMinFromBytes([]byte("OPTION /0123456789ABCDEF")))
	require.Equal(t, ErrPossibleHTTPRequest, h.DeserializeMinFromBytes([]byte("OPTION 0123456789ABCDEF")))
}
