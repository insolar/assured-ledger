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
	return proto.GetBoolExtension(message.Options, E_FieldsZeroable,
		proto.GetBoolExtension(file.Options, E_ZeroableAll, IsNotation(file, message)))
}

func IsNotation(file *descriptor.FileDescriptorProto, message *descriptor.DescriptorProto) bool {
	return proto.GetBoolExtension(message.Options, E_Notation,
		proto.GetBoolExtension(file.Options, E_NotationAll, false))
}

func GetStrFileExtension(file *descriptor.FileDescriptorProto, extension *proto.ExtensionDesc, defValue string) string {
	if file != nil && file.Options != nil {
		v, err := proto.GetExtension(file.Options, extension)
		if err == nil && v.(*string) != nil {
			return *(v.(*string))
		}
	}
	return defValue
}

func GetStrExtension(msg *descriptor.DescriptorProto, extension *proto.ExtensionDesc, defValue string) string {
	if msg != nil && msg.Options != nil {
		v, err := proto.GetExtension(msg.Options, extension)
		if err == nil && v.(*string) != nil {
			return *(v.(*string))
		}
	}
	return defValue
}

func GetStrFieldExtension(field *descriptor.FieldDescriptorProto, extension *proto.ExtensionDesc, defValue string) string {
	if field != nil && field.Options != nil {
		v, err := proto.GetExtension(field.Options, extension)
		if err == nil && v.(*string) != nil {
			return *(v.(*string))
		}
	}
	return defValue
}

func GetCustomContext(file *descriptor.FileDescriptorProto, msg *descriptor.DescriptorProto) string {
	return GetStrExtension(msg, E_Context, GetStrFileExtension(file, E_ContextAll, ""))
}

func GetCustomContextMethod(file *descriptor.FileDescriptorProto, msg *descriptor.DescriptorProto) string {
	return GetStrExtension(msg, E_ContextMethod, GetStrFileExtension(file, E_ContextMethodAll, "SetupContext"))
}

func IsCustomContext(file *descriptor.FileDescriptorProto, msg *descriptor.DescriptorProto) bool {
	return len(GetCustomContext(file, msg)) > 0
}

func GetCustomContextApply(field *descriptor.FieldDescriptorProto) string {
	return GetStrFieldExtension(field, E_CtxApply, "")
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

func IsRawBytes(field *descriptor.FieldDescriptorProto, defValue bool) bool {
	return proto.GetBoolExtension(field.Options, E_RawBytes, defValue)
}

func IsMappingForField(field *descriptor.FieldDescriptorProto, message *descriptor.DescriptorProto) bool {
	return proto.GetBoolExtension(field.Options, E_Mapping,
		proto.GetBoolExtension(message.Options, E_FieldsMapping, false))
}

func IsMappingForMessage(message *descriptor.DescriptorProto) bool {
	return proto.GetBoolExtension(message.Options, E_MessageMapping, false)
}

func GetCustomRegister(file *descriptor.FileDescriptorProto, msg *descriptor.DescriptorProto) string {
	return GetStrExtension(msg, E_Register, GetStrFileExtension(file, E_RegisterAll, ""))
}
