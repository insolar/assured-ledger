// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insproto

import (
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
)

func IsZeroable(field *descriptor.FieldDescriptorProto) bool {
	return proto.GetBoolExtension(field.Options, E_Zeroable, proto.GetBoolExtension(field.Options, E_InsprotoZeroableAll, true))
}

func IsNotation(file *descriptor.FileDescriptorProto, message *descriptor.DescriptorProto) bool {
	return proto.GetBoolExtension(message.Options, E_InsprotoNotation, proto.GetBoolExtension(file.Options, E_InsprotoNotationAll, true))
}

func GetCustomContext(field *descriptor.FieldDescriptorProto) string {
	if field == nil {
		return ""
	}
	if field.Options != nil {
		v, err := proto.GetExtension(field.Options, E_Context)
		if err == nil && v.(*string) != nil {
			return *(v.(*string))
		}
	}
	return ""
}

func IsCustomContext(field *descriptor.FieldDescriptorProto) bool {
	typ := GetCustomContext(field)
	return len(typ) > 0
}
