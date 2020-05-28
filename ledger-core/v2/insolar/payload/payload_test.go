// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload_test

import (
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
)

func TestPolymorphProducesExpectedBinary(t *testing.T) {
	pl := payload.Meta{}
	data, err := pl.Marshal()
	require.NoError(t, err)
	buf := proto.NewBuffer(data)

	_, err = buf.DecodeVarint()
	require.NoError(t, err)
	morph64, err := buf.DecodeVarint()
	require.NoError(t, err)

	require.Equal(t, uint32(payload.TypeMetaPolymorthID), uint32(morph64))
}

func _TestMarshalUnmarshalType(t *testing.T) {
	for _, expectedType := range payload.TypesMap {
		buf, err := payload.MarshalType(expectedType)
		require.NoError(t, err)

		tp, err := payload.UnmarshalType(buf)
		require.NoError(t, err)
		require.Equal(t, expectedType, tp)
	}
}

func _TestMarshalUnmarshal(t *testing.T) {
	for _, expectedType := range payload.TypesMap {
		if expectedType == payload.TypeUnknown {
			continue
		}

		typeBuf, err := payload.MarshalType(expectedType)
		require.NoError(t, err)
		pl, err := payload.Unmarshal(typeBuf)
		require.NoError(t, err, "Unmarshal() unknown type %s", expectedType.String())
		r := reflect.ValueOf(pl)
		f := reflect.Indirect(r).FieldByName("Polymorph")
		require.Equal(t, uint32(expectedType), uint32(f.Uint()), "Unmarshal() failed on type %s", expectedType.String())

		buf, err := payload.Marshal(pl)
		require.NoError(t, err, "Marshal() unknown type %s", expectedType.String())
		tp, err := payload.UnmarshalType(buf)
		require.NoError(t, err)
		require.Equal(t, expectedType, tp, "type mismatch between Marshal() and Unmarshal()")
	}
}
