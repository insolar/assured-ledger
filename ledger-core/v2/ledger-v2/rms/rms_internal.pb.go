// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: rms_internal.proto

package rms

import (
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	io "io"
	math "math"
	math_bits "math/bits"
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

type InternalRecordEnvelope struct {
	Head          interceptor     `protobuf:"bytes,17,opt,name=Head,proto3,customtype=interceptor" json:"Head"`
	Body          interceptorBody `protobuf:"bytes,18,opt,name=Body,proto3,customtype=interceptorBody" json:"Body"`
	BodySignature interceptor     `protobuf:"bytes,19,opt,name=BodySignature,proto3,customtype=interceptor" json:"BodySignature"`
	Extensions    []interceptor   `protobuf:"bytes,20,rep,name=Extensions,proto3,customtype=interceptor" json:"Extensions"`
}

func (m *InternalRecordEnvelope) Reset()         { *m = InternalRecordEnvelope{} }
func (m *InternalRecordEnvelope) String() string { return proto.CompactTextString(m) }
func (*InternalRecordEnvelope) ProtoMessage()    {}
func (*InternalRecordEnvelope) Descriptor() ([]byte, []int) {
	return fileDescriptor_4e39706ae3258ec3, []int{0}
}
func (m *InternalRecordEnvelope) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *InternalRecordEnvelope) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *InternalRecordEnvelope) XXX_Merge(src proto.Message) {
	xxx_messageInfo_InternalRecordEnvelope.Merge(m, src)
}
func (m *InternalRecordEnvelope) XXX_Size() int {
	return m.ProtoSize()
}
func (m *InternalRecordEnvelope) XXX_DiscardUnknown() {
	xxx_messageInfo_InternalRecordEnvelope.DiscardUnknown(m)
}

var xxx_messageInfo_InternalRecordEnvelope proto.InternalMessageInfo

type InternalMessageEnvelope struct {
	MsgBody           interceptor     `protobuf:"bytes,17,opt,name=MsgBody,proto3,customtype=interceptor" json:"MsgBody"`
	RecBody           interceptorBody `protobuf:"bytes,18,opt,name=RecBody,proto3,customtype=interceptorBody" json:"RecBody"`
	RecBodySignature  interceptor     `protobuf:"bytes,19,opt,name=RecBodySignature,proto3,customtype=interceptor" json:"RecBodySignature"`
	RecExtensions     []interceptor   `protobuf:"bytes,20,rep,name=RecExtensions,proto3,customtype=interceptor" json:"RecExtensions"`
	interceptorBundle `protobuf:"bytes,21,opt,name=BundleRecords,proto3,embedded=BundleRecords,customtype=interceptorBundle" json:"BundleRecords"`
}

func (m *InternalMessageEnvelope) Reset()         { *m = InternalMessageEnvelope{} }
func (m *InternalMessageEnvelope) String() string { return proto.CompactTextString(m) }
func (*InternalMessageEnvelope) ProtoMessage()    {}
func (*InternalMessageEnvelope) Descriptor() ([]byte, []int) {
	return fileDescriptor_4e39706ae3258ec3, []int{1}
}
func (m *InternalMessageEnvelope) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *InternalMessageEnvelope) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *InternalMessageEnvelope) XXX_Merge(src proto.Message) {
	xxx_messageInfo_InternalMessageEnvelope.Merge(m, src)
}
func (m *InternalMessageEnvelope) XXX_Size() int {
	return m.ProtoSize()
}
func (m *InternalMessageEnvelope) XXX_DiscardUnknown() {
	xxx_messageInfo_InternalMessageEnvelope.DiscardUnknown(m)
}

var xxx_messageInfo_InternalMessageEnvelope proto.InternalMessageInfo

type InternalMessageBundle struct {
	BundleRecords []InternalRecordEnvelope `protobuf:"bytes,20,rep,name=BundleRecords,proto3" json:"BundleRecords"`
}

func (m *InternalMessageBundle) Reset()         { *m = InternalMessageBundle{} }
func (m *InternalMessageBundle) String() string { return proto.CompactTextString(m) }
func (*InternalMessageBundle) ProtoMessage()    {}
func (*InternalMessageBundle) Descriptor() ([]byte, []int) {
	return fileDescriptor_4e39706ae3258ec3, []int{2}
}
func (m *InternalMessageBundle) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *InternalMessageBundle) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *InternalMessageBundle) XXX_Merge(src proto.Message) {
	xxx_messageInfo_InternalMessageBundle.Merge(m, src)
}
func (m *InternalMessageBundle) XXX_Size() int {
	return m.ProtoSize()
}
func (m *InternalMessageBundle) XXX_DiscardUnknown() {
	xxx_messageInfo_InternalMessageBundle.DiscardUnknown(m)
}

var xxx_messageInfo_InternalMessageBundle proto.InternalMessageInfo

type InternalRecordBody struct {
	// uint32 Polymorph = 16; not needed as field(17) has same properties and is never empty
	MainContent     GoGoMarshaller    `protobuf:"bytes,17,opt,name=MainContent,proto3,customtype=GoGoMarshaller" json:"MainContent"`
	ExtensionHashes []interceptorHash `protobuf:"bytes,18,rep,name=ExtensionHashes,proto3,customtype=interceptorHash" json:"ExtensionHashes,omitempty"`
}

func (m *InternalRecordBody) Reset()         { *m = InternalRecordBody{} }
func (m *InternalRecordBody) String() string { return proto.CompactTextString(m) }
func (*InternalRecordBody) ProtoMessage()    {}
func (*InternalRecordBody) Descriptor() ([]byte, []int) {
	return fileDescriptor_4e39706ae3258ec3, []int{3}
}
func (m *InternalRecordBody) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *InternalRecordBody) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *InternalRecordBody) XXX_Merge(src proto.Message) {
	xxx_messageInfo_InternalRecordBody.Merge(m, src)
}
func (m *InternalRecordBody) XXX_Size() int {
	return m.ProtoSize()
}
func (m *InternalRecordBody) XXX_DiscardUnknown() {
	xxx_messageInfo_InternalRecordBody.DiscardUnknown(m)
}

var xxx_messageInfo_InternalRecordBody proto.InternalMessageInfo

func init() {
	proto.RegisterType((*InternalRecordEnvelope)(nil), "rms.InternalRecordEnvelope")
	proto.RegisterType((*InternalMessageEnvelope)(nil), "rms.InternalMessageEnvelope")
	proto.RegisterType((*InternalMessageBundle)(nil), "rms.InternalMessageBundle")
	proto.RegisterType((*InternalRecordBody)(nil), "rms.InternalRecordBody")
}

func init() { proto.RegisterFile("rms_internal.proto", fileDescriptor_4e39706ae3258ec3) }

var fileDescriptor_4e39706ae3258ec3 = []byte{
	// 456 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x93, 0xd1, 0x6e, 0xd3, 0x30,
	0x14, 0x86, 0x63, 0x56, 0x31, 0x74, 0xca, 0x58, 0xe7, 0xb1, 0x2d, 0x1a, 0xc2, 0xad, 0x2a, 0x24,
	0x26, 0xa1, 0x75, 0x62, 0xbb, 0x81, 0x0b, 0x84, 0x14, 0x34, 0x75, 0x20, 0xf5, 0x26, 0x3c, 0xc0,
	0x70, 0xd3, 0x43, 0x1a, 0x29, 0xb5, 0x2b, 0xdb, 0x45, 0xf0, 0x16, 0x48, 0xbc, 0x00, 0x8f, 0xd3,
	0xcb, 0x5e, 0x4e, 0xbb, 0x88, 0xa0, 0xe1, 0x41, 0x50, 0x9c, 0x74, 0x4a, 0x02, 0x9b, 0x7a, 0x15,
	0x5b, 0xe7, 0xf3, 0x7f, 0xfc, 0xff, 0x27, 0x06, 0xaa, 0x26, 0xfa, 0x32, 0x12, 0x06, 0x95, 0xe0,
	0x71, 0x6f, 0xaa, 0xa4, 0x91, 0x74, 0x43, 0x4d, 0xf4, 0xe1, 0x71, 0x18, 0x99, 0xf1, 0x6c, 0xd8,
	0x0b, 0xe4, 0xe4, 0x24, 0x94, 0xa1, 0x3c, 0xb1, 0xb5, 0xe1, 0xec, 0xb3, 0xdd, 0xd9, 0x8d, 0x5d,
	0xe5, 0x67, 0xba, 0x7f, 0x08, 0xec, 0xbf, 0x2f, 0x64, 0x7c, 0x0c, 0xa4, 0x1a, 0x9d, 0x8b, 0x2f,
	0x18, 0xcb, 0x29, 0xd2, 0xe7, 0xd0, 0xb8, 0x40, 0x3e, 0x72, 0x77, 0x3a, 0xe4, 0xe8, 0xa1, 0xb7,
	0x3b, 0x4f, 0xda, 0xce, 0x75, 0xd2, 0x6e, 0xda, 0xa6, 0x01, 0x4e, 0x8d, 0x54, 0xbe, 0x05, 0xe8,
	0x0b, 0x68, 0x78, 0x72, 0xf4, 0xcd, 0xa5, 0x16, 0x3c, 0x28, 0xc0, 0xed, 0x12, 0x98, 0x95, 0x7d,
	0x0b, 0xd1, 0xd7, 0xb0, 0x95, 0x7d, 0x3f, 0x46, 0xa1, 0xe0, 0x66, 0xa6, 0xd0, 0xdd, 0xbd, 0x5d,
	0xbe, 0x4a, 0xd2, 0x33, 0x80, 0xf3, 0xaf, 0x06, 0x85, 0x8e, 0xa4, 0xd0, 0xee, 0xe3, 0xce, 0xc6,
	0x6d, 0xe7, 0x4a, 0xd8, 0x87, 0xc6, 0x03, 0xd2, 0x6a, 0x75, 0x93, 0x7b, 0x70, 0xb0, 0xb2, 0x39,
	0x40, 0xad, 0x79, 0x88, 0x37, 0x3e, 0x8f, 0x61, 0x73, 0xa0, 0x43, 0xeb, 0xe0, 0x0e, 0xab, 0x2b,
	0x86, 0xbe, 0x84, 0x4d, 0x1f, 0x83, 0x75, 0x0c, 0xaf, 0x38, 0xfa, 0x16, 0x5a, 0xc5, 0x72, 0x2d,
	0xdb, 0xff, 0xc0, 0x59, 0x68, 0x3e, 0x06, 0xeb, 0x99, 0xaf, 0x92, 0xf4, 0x12, 0xb6, 0xbc, 0x99,
	0x18, 0xc5, 0x98, 0x4f, 0x57, 0xbb, 0x7b, 0x1d, 0x72, 0xd4, 0x3c, 0x3d, 0xec, 0xa9, 0x89, 0xee,
	0xd5, 0x22, 0xc9, 0x41, 0xef, 0x69, 0x26, 0xbb, 0x48, 0xda, 0xe4, 0x3a, 0x69, 0xef, 0x94, 0x4d,
	0xe5, 0x3a, 0x55, 0xbd, 0x22, 0xe0, 0x4f, 0xb0, 0xf7, 0x5f, 0x31, 0xda, 0xaf, 0xf7, 0xcf, 0xae,
	0xde, 0x3c, 0x7d, 0x52, 0xe9, 0x5f, 0xfd, 0xf3, 0xbc, 0x46, 0x76, 0x81, 0x5a, 0x9f, 0xee, 0x0f,
	0x02, 0xb4, 0xca, 0xdb, 0x6c, 0x5f, 0x41, 0x73, 0xc0, 0x23, 0xf1, 0x4e, 0x0a, 0x83, 0xc2, 0x14,
	0x13, 0xdc, 0x2f, 0x82, 0x79, 0xd4, 0x97, 0x7d, 0x39, 0xe0, 0x4a, 0x8f, 0x79, 0x1c, 0xa3, 0xf2,
	0xcb, 0x28, 0x7d, 0x03, 0xdb, 0x37, 0x39, 0x5d, 0x70, 0x3d, 0x46, 0xed, 0xd2, 0x3c, 0xd6, 0xda,
	0x30, 0xb3, 0xa2, 0x5f, 0x67, 0x73, 0xdf, 0xde, 0xb3, 0xf9, 0x6f, 0xe6, 0xcc, 0x97, 0x8c, 0x2c,
	0x96, 0x8c, 0x5c, 0x2d, 0x19, 0xf9, 0xb5, 0x64, 0xce, 0xf7, 0x94, 0x39, 0x3f, 0x53, 0x46, 0x16,
	0x29, 0x73, 0xae, 0x52, 0xe6, 0x0c, 0xef, 0xdb, 0xc7, 0x76, 0xf6, 0x37, 0x00, 0x00, 0xff, 0xff,
	0x7b, 0xef, 0x8b, 0x8f, 0xb6, 0x03, 0x00, 0x00,
}

func (m *InternalRecordEnvelope) Marshal() (dAtA []byte, err error) {
	size := m.ProtoSize()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *InternalRecordEnvelope) MarshalTo(dAtA []byte) (int, error) {
	size := m.ProtoSize()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *InternalRecordEnvelope) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Extensions) > 0 {
		for iNdEx := len(m.Extensions) - 1; iNdEx >= 0; iNdEx-- {
			{
				size := m.Extensions[iNdEx].ProtoSize()
				i -= size
				if _, err := m.Extensions[iNdEx].MarshalTo(dAtA[i:]); err != nil {
					return 0, err
				}
				i = encodeVarintRmsInternal(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x1
			i--
			dAtA[i] = 0xa2
		}
	}
	{
		size := m.BodySignature.ProtoSize()
		i -= size
		if _, err := m.BodySignature.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintRmsInternal(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1
	i--
	dAtA[i] = 0x9a
	{
		size := m.Body.ProtoSize()
		i -= size
		if _, err := m.Body.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintRmsInternal(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1
	i--
	dAtA[i] = 0x92
	{
		size := m.Head.ProtoSize()
		i -= size
		if _, err := m.Head.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintRmsInternal(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1
	i--
	dAtA[i] = 0x8a
	return len(dAtA) - i, nil
}

func (m *InternalMessageEnvelope) Marshal() (dAtA []byte, err error) {
	size := m.ProtoSize()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *InternalMessageEnvelope) MarshalTo(dAtA []byte) (int, error) {
	size := m.ProtoSize()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *InternalMessageEnvelope) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size := m.interceptorBundle.ProtoSize()
		i -= size
		if _, err := m.interceptorBundle.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintRmsInternal(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1
	i--
	dAtA[i] = 0xaa
	if len(m.RecExtensions) > 0 {
		for iNdEx := len(m.RecExtensions) - 1; iNdEx >= 0; iNdEx-- {
			{
				size := m.RecExtensions[iNdEx].ProtoSize()
				i -= size
				if _, err := m.RecExtensions[iNdEx].MarshalTo(dAtA[i:]); err != nil {
					return 0, err
				}
				i = encodeVarintRmsInternal(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x1
			i--
			dAtA[i] = 0xa2
		}
	}
	{
		size := m.RecBodySignature.ProtoSize()
		i -= size
		if _, err := m.RecBodySignature.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintRmsInternal(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1
	i--
	dAtA[i] = 0x9a
	{
		size := m.RecBody.ProtoSize()
		i -= size
		if _, err := m.RecBody.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintRmsInternal(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1
	i--
	dAtA[i] = 0x92
	{
		size := m.MsgBody.ProtoSize()
		i -= size
		if _, err := m.MsgBody.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintRmsInternal(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1
	i--
	dAtA[i] = 0x8a
	return len(dAtA) - i, nil
}

func (m *InternalMessageBundle) Marshal() (dAtA []byte, err error) {
	size := m.ProtoSize()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *InternalMessageBundle) MarshalTo(dAtA []byte) (int, error) {
	size := m.ProtoSize()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *InternalMessageBundle) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.BundleRecords) > 0 {
		for iNdEx := len(m.BundleRecords) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.BundleRecords[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintRmsInternal(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x1
			i--
			dAtA[i] = 0xa2
		}
	}
	return len(dAtA) - i, nil
}

func (m *InternalRecordBody) Marshal() (dAtA []byte, err error) {
	size := m.ProtoSize()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *InternalRecordBody) MarshalTo(dAtA []byte) (int, error) {
	size := m.ProtoSize()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *InternalRecordBody) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.ExtensionHashes) > 0 {
		for iNdEx := len(m.ExtensionHashes) - 1; iNdEx >= 0; iNdEx-- {
			{
				size := m.ExtensionHashes[iNdEx].ProtoSize()
				i -= size
				if _, err := m.ExtensionHashes[iNdEx].MarshalTo(dAtA[i:]); err != nil {
					return 0, err
				}
				i = encodeVarintRmsInternal(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x1
			i--
			dAtA[i] = 0x92
		}
	}
	{
		size := m.MainContent.ProtoSize()
		i -= size
		if _, err := m.MainContent.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintRmsInternal(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1
	i--
	dAtA[i] = 0x8a
	return len(dAtA) - i, nil
}

func encodeVarintRmsInternal(dAtA []byte, offset int, v uint64) int {
	offset -= sovRmsInternal(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *InternalRecordEnvelope) ProtoSize() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.Head.ProtoSize()
	n += 2 + l + sovRmsInternal(uint64(l))
	l = m.Body.ProtoSize()
	n += 2 + l + sovRmsInternal(uint64(l))
	l = m.BodySignature.ProtoSize()
	n += 2 + l + sovRmsInternal(uint64(l))
	if len(m.Extensions) > 0 {
		for _, e := range m.Extensions {
			l = e.ProtoSize()
			n += 2 + l + sovRmsInternal(uint64(l))
		}
	}
	return n
}

func (m *InternalMessageEnvelope) ProtoSize() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.MsgBody.ProtoSize()
	n += 2 + l + sovRmsInternal(uint64(l))
	l = m.RecBody.ProtoSize()
	n += 2 + l + sovRmsInternal(uint64(l))
	l = m.RecBodySignature.ProtoSize()
	n += 2 + l + sovRmsInternal(uint64(l))
	if len(m.RecExtensions) > 0 {
		for _, e := range m.RecExtensions {
			l = e.ProtoSize()
			n += 2 + l + sovRmsInternal(uint64(l))
		}
	}
	l = m.interceptorBundle.ProtoSize()
	n += 2 + l + sovRmsInternal(uint64(l))
	return n
}

func (m *InternalMessageBundle) ProtoSize() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.BundleRecords) > 0 {
		for _, e := range m.BundleRecords {
			l = e.ProtoSize()
			n += 2 + l + sovRmsInternal(uint64(l))
		}
	}
	return n
}

func (m *InternalRecordBody) ProtoSize() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.MainContent.ProtoSize()
	n += 2 + l + sovRmsInternal(uint64(l))
	if len(m.ExtensionHashes) > 0 {
		for _, e := range m.ExtensionHashes {
			l = e.ProtoSize()
			n += 2 + l + sovRmsInternal(uint64(l))
		}
	}
	return n
}

func sovRmsInternal(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozRmsInternal(x uint64) (n int) {
	return sovRmsInternal(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *InternalRecordEnvelope) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowRmsInternal
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: InternalRecordEnvelope: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: InternalRecordEnvelope: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 17:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Head", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRmsInternal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthRmsInternal
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthRmsInternal
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Head.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 18:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Body", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRmsInternal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthRmsInternal
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthRmsInternal
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Body.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 19:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field BodySignature", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRmsInternal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthRmsInternal
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthRmsInternal
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.BodySignature.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 20:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Extensions", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRmsInternal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthRmsInternal
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthRmsInternal
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			var v interceptor
			m.Extensions = append(m.Extensions, v)
			if err := m.Extensions[len(m.Extensions)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipRmsInternal(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthRmsInternal
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthRmsInternal
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *InternalMessageEnvelope) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowRmsInternal
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: InternalMessageEnvelope: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: InternalMessageEnvelope: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 17:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MsgBody", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRmsInternal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthRmsInternal
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthRmsInternal
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.MsgBody.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 18:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field RecBody", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRmsInternal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthRmsInternal
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthRmsInternal
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.RecBody.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 19:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field RecBodySignature", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRmsInternal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthRmsInternal
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthRmsInternal
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.RecBodySignature.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 20:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field RecExtensions", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRmsInternal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthRmsInternal
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthRmsInternal
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			var v interceptor
			m.RecExtensions = append(m.RecExtensions, v)
			if err := m.RecExtensions[len(m.RecExtensions)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 21:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field interceptorBundle", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRmsInternal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthRmsInternal
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthRmsInternal
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.interceptorBundle.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipRmsInternal(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthRmsInternal
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthRmsInternal
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *InternalMessageBundle) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowRmsInternal
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: InternalMessageBundle: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: InternalMessageBundle: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 20:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field BundleRecords", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRmsInternal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthRmsInternal
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthRmsInternal
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.BundleRecords = append(m.BundleRecords, InternalRecordEnvelope{})
			if err := m.BundleRecords[len(m.BundleRecords)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipRmsInternal(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthRmsInternal
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthRmsInternal
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *InternalRecordBody) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowRmsInternal
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: InternalRecordBody: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: InternalRecordBody: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 17:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MainContent", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRmsInternal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthRmsInternal
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthRmsInternal
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.MainContent.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 18:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ExtensionHashes", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRmsInternal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthRmsInternal
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthRmsInternal
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			var v interceptorHash
			m.ExtensionHashes = append(m.ExtensionHashes, v)
			if err := m.ExtensionHashes[len(m.ExtensionHashes)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipRmsInternal(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthRmsInternal
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthRmsInternal
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipRmsInternal(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowRmsInternal
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowRmsInternal
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowRmsInternal
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthRmsInternal
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupRmsInternal
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthRmsInternal
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthRmsInternal        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowRmsInternal          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupRmsInternal = fmt.Errorf("proto: unexpected end of group")
)
