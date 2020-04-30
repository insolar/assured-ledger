// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: ins.proto

package insproto

import (
	fmt "fmt"
	proto "github.com/gogo/protobuf/proto"
	descriptor "github.com/golang/protobuf/protoc-gen-go/descriptor"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

var E_InsprotoNotationAll = &proto.ExtensionDesc{
	ExtendedType:  (*descriptor.FileOptions)(nil),
	ExtensionType: (*bool)(nil),
	Field:         63901,
	Name:          "insproto.insproto_notation_all",
	Tag:           "varint,63901,opt,name=insproto_notation_all",
	Filename:      "ins.proto",
}

var E_InsprotoZeroableAll = &proto.ExtensionDesc{
	ExtendedType:  (*descriptor.FileOptions)(nil),
	ExtensionType: (*bool)(nil),
	Field:         63902,
	Name:          "insproto.insproto_zeroable_all",
	Tag:           "varint,63902,opt,name=insproto_zeroable_all",
	Filename:      "ins.proto",
}

var E_InsprotoNotation = &proto.ExtensionDesc{
	ExtendedType:  (*descriptor.MessageOptions)(nil),
	ExtensionType: (*bool)(nil),
	Field:         64901,
	Name:          "insproto.insproto_notation",
	Tag:           "varint,64901,opt,name=insproto_notation",
	Filename:      "ins.proto",
}

var E_InsprotoZeroable = &proto.ExtensionDesc{
	ExtendedType:  (*descriptor.MessageOptions)(nil),
	ExtensionType: (*bool)(nil),
	Field:         64902,
	Name:          "insproto.insproto_zeroable",
	Tag:           "varint,64902,opt,name=insproto_zeroable",
	Filename:      "ins.proto",
}

var E_Context = &proto.ExtensionDesc{
	ExtendedType:  (*descriptor.FieldOptions)(nil),
	ExtensionType: (*string)(nil),
	Field:         65901,
	Name:          "insproto.context",
	Tag:           "bytes,65901,opt,name=context",
	Filename:      "ins.proto",
}

var E_Zeroable = &proto.ExtensionDesc{
	ExtendedType:  (*descriptor.FieldOptions)(nil),
	ExtensionType: (*bool)(nil),
	Field:         65902,
	Name:          "insproto.zeroable",
	Tag:           "varint,65902,opt,name=zeroable",
	Filename:      "ins.proto",
}

func init() {
	proto.RegisterExtension(E_InsprotoNotationAll)
	proto.RegisterExtension(E_InsprotoZeroableAll)
	proto.RegisterExtension(E_InsprotoNotation)
	proto.RegisterExtension(E_InsprotoZeroable)
	proto.RegisterExtension(E_Context)
	proto.RegisterExtension(E_Zeroable)
}

func init() { proto.RegisterFile("ins.proto", fileDescriptor_faf174f739482dca) }

var fileDescriptor_faf174f739482dca = []byte{
	// 257 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0xcc, 0xcc, 0x2b, 0xd6,
	0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0xe2, 0xc8, 0xcc, 0x2b, 0x06, 0xb3, 0xa4, 0x14, 0xd2, 0xf3,
	0xf3, 0xd3, 0x73, 0x52, 0xf5, 0xc1, 0xbc, 0xa4, 0xd2, 0x34, 0xfd, 0x94, 0xd4, 0xe2, 0xe4, 0xa2,
	0xcc, 0x82, 0x92, 0xfc, 0x22, 0x88, 0x5a, 0xab, 0x20, 0x2e, 0x51, 0x98, 0xea, 0xf8, 0xbc, 0xfc,
	0x92, 0xc4, 0x92, 0xcc, 0xfc, 0xbc, 0xf8, 0xc4, 0x9c, 0x1c, 0x21, 0x19, 0x3d, 0x88, 0x5e, 0x3d,
	0x98, 0x5e, 0x3d, 0xb7, 0xcc, 0x9c, 0x54, 0xff, 0x02, 0x90, 0x82, 0x62, 0x89, 0xb9, 0x9f, 0x99,
	0x15, 0x18, 0x35, 0x38, 0x82, 0x84, 0x61, 0x9a, 0xfd, 0xa0, 0x7a, 0x1d, 0x73, 0x72, 0x50, 0xcc,
	0xac, 0x4a, 0x2d, 0xca, 0x4f, 0x4c, 0xca, 0x49, 0x25, 0xc2, 0xcc, 0x79, 0xe8, 0x66, 0x46, 0x41,
	0xf5, 0x82, 0xcc, 0xf4, 0xe3, 0x12, 0xc4, 0x70, 0xa7, 0x90, 0x3c, 0x86, 0x79, 0xbe, 0xa9, 0xc5,
	0xc5, 0x89, 0xe9, 0x70, 0x23, 0x5b, 0x7f, 0x43, 0x8c, 0x14, 0x40, 0x77, 0x26, 0x8a, 0x79, 0x30,
	0x37, 0x12, 0x36, 0xaf, 0x0d, 0xdd, 0x3c, 0x98, 0x13, 0xad, 0x2c, 0xb9, 0xd8, 0x93, 0xf3, 0xf3,
	0x4a, 0x52, 0x2b, 0x4a, 0x84, 0x64, 0xb1, 0xf8, 0x32, 0x35, 0x27, 0x05, 0x66, 0xc6, 0xdb, 0x26,
	0x16, 0x05, 0x46, 0x0d, 0xce, 0x20, 0x98, 0x7a, 0x2b, 0x6b, 0x2e, 0x0e, 0xb8, 0x0b, 0x08, 0xe8,
	0x7d, 0x07, 0xd6, 0xcb, 0x11, 0x04, 0xd7, 0xe0, 0x24, 0x71, 0xe2, 0x91, 0x1c, 0xe3, 0x85, 0x47,
	0x72, 0x8c, 0x0f, 0x1e, 0xc9, 0x31, 0x4e, 0x78, 0x2c, 0xc7, 0x70, 0xe1, 0xb1, 0x1c, 0xc3, 0x8d,
	0xc7, 0x72, 0x0c, 0x80, 0x00, 0x00, 0x00, 0xff, 0xff, 0xab, 0xcf, 0xbf, 0xd3, 0x11, 0x02, 0x00,
	0x00,
}
