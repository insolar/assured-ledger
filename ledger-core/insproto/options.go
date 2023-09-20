package insproto

import (
	"strings"

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

func IsNullableAll(file *descriptor.FileDescriptorProto) bool {
	return proto.GetBoolExtension(file.Options, E_NullableAll,
		!proto.GetBoolExtension(file.Options, E_NotationAll, false))
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

func GetCustomMessageContextApply(file *descriptor.FileDescriptorProto, msg *descriptor.DescriptorProto) string {
	return GetStrExtension(msg, E_MessageCtxApply, GetStrFileExtension(file, E_MessageCtxApplyAll, ""))
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

func GetProjectionID(msg *descriptor.DescriptorProto) uint64 {
	if msg != nil && msg.Options != nil {
		v, err := proto.GetExtension(msg.Options, E_ProjectionId)
		if err == nil && v.(*uint64) != nil {
			return *(v.(*uint64))
		}
	}
	return 0
}

func IsProjection(file *descriptor.FileDescriptorProto, msg *descriptor.DescriptorProto) bool {
	names := `,` + GetStrFileExtension(file, E_ProjectionNames, "") + `,`
	return proto.GetBoolExtension(msg.Options, E_Projection, strings.Contains(names, `,`+msg.GetName()+`,`))
}

func IsRawBytes(field *descriptor.FieldDescriptorProto, defValue bool) bool {
	return proto.GetBoolExtension(field.Options, E_RawBytes, defValue)
}

func IsMappingForField(field *descriptor.FieldDescriptorProto, message *descriptor.DescriptorProto, file *descriptor.FileDescriptorProto) bool {
	return proto.GetBoolExtension(field.Options, E_Mapping,
		proto.GetBoolExtension(message.Options, E_FieldsMapping,
			proto.GetBoolExtension(file.Options, E_FieldsMappingAll, false)))
}

func IsMappingForMessage(file *descriptor.FileDescriptorProto, message *descriptor.DescriptorProto) bool {
	return proto.GetBoolExtension(message.Options, E_MessageMapping,
		proto.GetBoolExtension(file.Options, E_MessageMappingAll, false))
}

func GetCustomRegister(file *descriptor.FileDescriptorProto, msg *descriptor.DescriptorProto) string {
	return GetStrExtension(msg, E_Register, GetStrFileExtension(file, E_RegisterAll, ""))
}
