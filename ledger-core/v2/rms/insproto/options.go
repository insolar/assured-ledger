// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insproto

import (
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
)

func IsZeroable(field *descriptor.FieldDescriptorProto, defValue bool) bool {
	return proto.GetBoolExtension(field.Options, E_Zeroable, defValue)
}

func IsZeroableDefault(file *descriptor.FileDescriptorProto, message *descriptor.DescriptorProto) bool {
	return proto.GetBoolExtension(message.Options, E_InsprotoZeroable,
		proto.GetBoolExtension(file.Options, E_InsprotoZeroableAll, IsNotation(file, message)))
}

func IsNotation(file *descriptor.FileDescriptorProto, message *descriptor.DescriptorProto) bool {
	return proto.GetBoolExtension(message.Options, E_InsprotoNotation,
		proto.GetBoolExtension(file.Options, E_InsprotoNotationAll, false))
}

func GetCustomContext(msg *descriptor.DescriptorProto) string {
	if msg != nil && msg.Options != nil {
		v, err := proto.GetExtension(msg.Options, E_Context)
		if err == nil && v.(*string) != nil {
			return *(v.(*string))
		}
	}
	return ""
}

func GetCustomContextMethod(msg *descriptor.DescriptorProto) string {
	if msg != nil && msg.Options != nil {
		v, err := proto.GetExtension(msg.Options, E_ContextMethod)
		if err == nil && v.(*string) != nil {
			return *(v.(*string))
		}
	}
	return "SetupContext"
}

func IsCustomContext(msg *descriptor.DescriptorProto) bool {
	return len(GetCustomContext(msg)) > 0
}

func GetCustomContextApply(field *descriptor.FieldDescriptorProto) string {
	if field != nil && field.Options != nil {
		v, err := proto.GetExtension(field.Options, E_CtxApply)
		if err == nil && v.(*string) != nil {
			return *(v.(*string))
		}
	}
	return ""
}

func IsCustomContextApply(field *descriptor.FieldDescriptorProto) bool {
	return len(GetCustomContextApply(field)) > 0
}

func GetPolymorphID(msg *descriptor.DescriptorProto) uint64 {
	if msg != nil && msg.Options != nil {
		v, err := proto.GetExtension(msg.Options, E_Id)
		if err == nil && v.(*uint64) != nil {
			return *(v.(*uint64))
		}
	}
	return 0
}

func HasPolymorphID(msg *descriptor.DescriptorProto) bool {
	if msg != nil && msg.Options != nil {
		v, err := proto.GetExtension(msg.Options, E_Id)
		if err == nil && v.(*uint64) != nil {
			return true
		}
	}
	return false
}

func IsHead(msg *descriptor.DescriptorProto) bool {
	return proto.GetBoolExtension(msg.Options, E_Head, msg.GetName() == `Head`)
}
