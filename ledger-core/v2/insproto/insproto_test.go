// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insproto

import (
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
)

func TestFieldMapFQN(t *testing.T) {
	name := proto.MessageName(&FieldMap{})
	require.Equal(t, `.`+name, FieldMapFQN)
	require.Equal(t, FieldMapPackage, reflect.TypeOf(FieldMap{}).PkgPath())
}
