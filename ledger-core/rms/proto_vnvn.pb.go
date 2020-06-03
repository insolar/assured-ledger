// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: proto_vnvn.proto

package rms

import (
	bytes "bytes"
	fmt "fmt"
	io "io"
	math "math"
	math_bits "math/bits"

	_ "github.com/gogo/protobuf/gogoproto"
	github_com_gogo_protobuf_proto "github.com/gogo/protobuf/proto"
	proto "github.com/gogo/protobuf/proto"
	_ "github.com/insolar/assured-ledger/ledger-core/insproto"
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

type MessageExample struct {
	RecordExample `protobuf:"bytes,19,opt,name=Record,proto3,embedded=Record" json:"Record"`
	// Add here custom fields
	MsgParam uint64 `protobuf:"varint,1800,opt,name=MsgParam,proto3" json:"MsgParam"`
	MsgBytes []byte `protobuf:"bytes,1801,opt,name=MsgBytes,proto3" json:"MsgBytes"`
}

func (m *MessageExample) Reset()         { *m = MessageExample{} }
func (m *MessageExample) String() string { return proto.CompactTextString(m) }
func (*MessageExample) ProtoMessage()    {}
func (*MessageExample) Descriptor() ([]byte, []int) {
	return fileDescriptor_fc353dc03c844eae, []int{0}
}
func (m *MessageExample) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MessageExample) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *MessageExample) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MessageExample.Merge(m, src)
}
func (m *MessageExample) XXX_Size() int {
	return m.ProtoSize()
}
func (m *MessageExample) XXX_DiscardUnknown() {
	xxx_messageInfo_MessageExample.DiscardUnknown(m)
}

var xxx_messageInfo_MessageExample proto.InternalMessageInfo

func (m *MessageExample) GetMsgParam() uint64 {
	if m != nil {
		return m.MsgParam
	}
	return 0
}

func (m *MessageExample) GetMsgBytes() []byte {
	if m != nil {
		return m.MsgBytes
	}
	return nil
}

func (m *MessageExample_Head) Reset()         { *m = MessageExample_Head{} }
func (m *MessageExample_Head) String() string { return proto.CompactTextString(m) }
func (*MessageExample_Head) ProtoMessage()    {}
func (*MessageExample_Head) Descriptor() ([]byte, []int) {
	return fileDescriptor_fc353dc03c844eae, []int{0, 0}
}
func (m *MessageExample_Head) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MessageExample_Head) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *MessageExample_Head) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MessageExample_Head.Merge(m, src)
}
func (m *MessageExample_Head) XXX_Size() int {
	return m.ProtoSize()
}
func (m *MessageExample_Head) XXX_DiscardUnknown() {
	xxx_messageInfo_MessageExample_Head.DiscardUnknown(m)
}

var xxx_messageInfo_MessageExample_Head proto.InternalMessageInfo

func init() {
	proto.RegisterType((*MessageExample)(nil), "rms.MessageExample")
	proto.RegisterType((*MessageExample_Head)(nil), "rms.MessageExample.Head")
}

func init() { proto.RegisterFile("proto_vnvn.proto", fileDescriptor_fc353dc03c844eae) }

var fileDescriptor_fc353dc03c844eae = []byte{
	// 435 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x91, 0xb1, 0x6b, 0xdb, 0x40,
	0x18, 0xc5, 0x75, 0xb5, 0x9a, 0x2a, 0x4a, 0x30, 0x46, 0x4a, 0x40, 0x78, 0xb8, 0x73, 0xdb, 0x25,
	0x8b, 0x2d, 0x48, 0x37, 0x6f, 0x55, 0x28, 0x94, 0x80, 0xa1, 0x28, 0xdd, 0xcb, 0xd9, 0xbe, 0x5e,
	0x04, 0x96, 0x4e, 0xdc, 0x9d, 0x4c, 0xfc, 0x1f, 0xb8, 0x9d, 0x02, 0x85, 0x42, 0x3c, 0xb9, 0x9b,
	0xe9, 0xd8, 0xbf, 0xa0, 0xd0, 0xc5, 0xa3, 0xe9, 0xa4, 0xa1, 0x84, 0x56, 0x5a, 0x0a, 0x5d, 0x4a,
	0x87, 0x0e, 0x99, 0x8a, 0x4e, 0xe7, 0x12, 0x3a, 0x75, 0xd2, 0xf7, 0xbd, 0xef, 0xf7, 0x1e, 0x4f,
	0x92, 0xdd, 0x4a, 0x39, 0x93, 0xec, 0xc5, 0x34, 0x99, 0x26, 0x3d, 0x35, 0x3a, 0x0d, 0x1e, 0x8b,
	0x76, 0x97, 0x46, 0xf2, 0x3c, 0x1b, 0xf6, 0x46, 0x2c, 0xf6, 0x29, 0xa3, 0xcc, 0x57, 0xb7, 0x61,
	0xf6, 0x52, 0x6d, 0x6a, 0x51, 0x53, 0xed, 0x69, 0x9f, 0xdc, 0xc2, 0xa3, 0x44, 0xb0, 0x09, 0xe6,
	0x3e, 0x16, 0x22, 0xe3, 0x64, 0xdc, 0x9d, 0x90, 0x31, 0x25, 0xdc, 0xaf, 0x1f, 0xdd, 0x11, 0xe3,
	0xc4, 0x9f, 0x1e, 0x57, 0x54, 0x9d, 0x12, 0x25, 0x42, 0x87, 0xec, 0xf2, 0x78, 0x3b, 0xba, 0x75,
	0x2b, 0x4e, 0x46, 0x8c, 0x8f, 0xb5, 0xf8, 0xe0, 0xd3, 0x1d, 0xbb, 0x39, 0x20, 0x42, 0x60, 0x4a,
	0x9e, 0x5c, 0xe0, 0x38, 0x9d, 0x10, 0xe7, 0xb1, 0xbd, 0x13, 0x2a, 0xc6, 0x73, 0x3b, 0xe0, 0x68,
	0xef, 0xd8, 0xe9, 0x55, 0x19, 0xb5, 0xa4, 0x99, 0xe0, 0x70, 0x7d, 0x8d, 0x8c, 0xcd, 0x35, 0x02,
	0x37, 0x6f, 0x3b, 0xbb, 0x03, 0x41, 0xeb, 0x6b, 0xa8, 0x8d, 0xce, 0x7d, 0xdb, 0x1a, 0x08, 0xfa,
	0x0c, 0x73, 0x1c, 0x7b, 0xf3, 0x66, 0x07, 0x1c, 0x99, 0x81, 0x59, 0x39, 0xc2, 0xbf, 0xb2, 0x46,
	0x82, 0x99, 0x24, 0xc2, 0x7b, 0x55, 0x21, 0xfb, 0xb7, 0x10, 0x25, 0xb7, 0x2f, 0x81, 0x6d, 0x3e,
	0x25, 0xf8, 0xbf, 0xe2, 0x1e, 0xda, 0x8d, 0x33, 0xc9, 0xbd, 0x43, 0xd5, 0x78, 0x4f, 0x35, 0x0e,
	0xa2, 0x04, 0xf3, 0x99, 0x26, 0xab, 0x6b, 0xbf, 0x3f, 0x5f, 0x22, 0x63, 0xb5, 0x44, 0xe0, 0xe7,
	0x3b, 0x64, 0xac, 0x72, 0x04, 0x3e, 0xe6, 0xe8, 0xf7, 0x9b, 0x1f, 0x5f, 0x1a, 0x9f, 0x73, 0xb4,
	0x7f, 0x46, 0x64, 0x96, 0x9e, 0xb0, 0x44, 0x92, 0x0b, 0xf9, 0x2b, 0x47, 0xdb, 0xaf, 0xa2, 0x95,
	0x53, 0xd3, 0x02, 0xad, 0x83, 0xbe, 0xb5, 0x75, 0xa8, 0xdd, 0x3d, 0xbd, 0x6b, 0x1d, 0xb4, 0xe6,
	0xcd, 0xe0, 0x7c, 0xfd, 0x0d, 0x82, 0x55, 0x01, 0xc1, 0xba, 0x80, 0x60, 0x53, 0x40, 0x90, 0x17,
	0x10, 0x7c, 0x2d, 0xa0, 0x71, 0x59, 0x42, 0x63, 0x59, 0x42, 0xb0, 0x29, 0xa1, 0x91, 0x97, 0xd0,
	0xf8, 0x7e, 0x85, 0xc0, 0xcd, 0x15, 0xba, 0xa7, 0xc3, 0x5f, 0x2f, 0x90, 0x7a, 0xc3, 0xf7, 0x0b,
	0xe4, 0x86, 0x84, 0x46, 0x42, 0x12, 0xae, 0x4f, 0xcf, 0x67, 0x29, 0xf9, 0xb0, 0xf8, 0xb7, 0xc6,
	0x70, 0x47, 0xfd, 0xb6, 0x47, 0x7f, 0x02, 0x00, 0x00, 0xff, 0xff, 0x55, 0xf0, 0xc1, 0x8a, 0x63,
	0x02, 0x00, 0x00,
}

func (this *MessageExample) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*MessageExample)
	if !ok {
		that2, ok := that.(MessageExample)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if !this.RecordExample.Equal(&that1.RecordExample) {
		return false
	}
	if this.MsgParam != that1.MsgParam {
		return false
	}
	if !bytes.Equal(this.MsgBytes, that1.MsgBytes) {
		return false
	}
	return true
}
func (this *MessageExample_Head) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*MessageExample_Head)
	if !ok {
		that2, ok := that.(MessageExample_Head)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if !this.Str.Equal(&that1.Str) {
		return false
	}
	if this.MsgParam != that1.MsgParam {
		return false
	}
	return true
}

type MessageExample_HeadFace interface {
	Proto() github_com_gogo_protobuf_proto.Message
	GetStr() Binary
	GetMsgParam() uint64
}

func (this *MessageExample_Head) Proto() github_com_gogo_protobuf_proto.Message {
	return this
}

func (this *MessageExample_Head) TestProto() github_com_gogo_protobuf_proto.Message {
	return NewMessageExample_HeadFromFace(this)
}

func (this *MessageExample_Head) GetStr() Binary {
	return this.Str
}

func (this *MessageExample_Head) GetMsgParam() uint64 {
	return this.MsgParam
}

func NewMessageExample_HeadFromFace(that MessageExample_HeadFace) *MessageExample_Head {
	this := &MessageExample_Head{}
	this.Str = that.GetStr()
	this.MsgParam = that.GetMsgParam()
	return this
}

type MessageExampleHead MessageExample_HeadFace
type MessageExample_Head MessageExample

func (m *MessageExample) AsHead() *MessageExample_Head {
	return (*MessageExample_Head)(m)
}

func (m *MessageExample) AsHeadFace() MessageExampleHead {
	if m == nil {
		return nil
	}
	return (*MessageExample_Head)(m)
}

func (m *MessageExample_Head) AsMessageExample() *MessageExample {
	return (*MessageExample)(m)
}

func (m *MessageExample_Head) AsProjectionBase() interface{} {
	if m == nil {
		return nil
	}
	return (*MessageExample)(m)
}

func (m *MessageExample) AsProjection(name string) interface{} {
	if m == nil {
		return nil
	}
	switch name {
	case "Head":
		return m.AsHead()
	}
	return nil
}

func (m *MessageExample) SetupContext(ctx MessageContext) error {
	if err := ctx.MsgRecord(m, 19, &m.RecordExample); err != nil {
		return err
	}
	return ctx.Message(m, 999999990)
}

const TypeMessageExamplePolymorthID uint64 = 999999990

func (*MessageExample) GetDefaultPolymorphID() uint64 {
	return 999999990
}

func (m *MessageExample) Marshal() (dAtA []byte, err error) {
	size := m.ProtoSize()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MessageExample) MarshalTo(dAtA []byte) (int, error) {
	size := m.ProtoSize()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MessageExample) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l, fieldEnd int
	_, _ = l, fieldEnd
	if len(m.MsgBytes) > 0 {
		i -= len(m.MsgBytes)
		copy(dAtA[i:], m.MsgBytes)
		i--
		dAtA[i] = 132
		i = encodeVarintProtoVnvn(dAtA, i, uint64(len(m.MsgBytes)+1))
		i--
		dAtA[i] = 0x70
		i--
		dAtA[i] = 0xca
	}
	if m.MsgParam != 0 {
		i = encodeVarintProtoVnvn(dAtA, i, uint64(m.MsgParam))
		i--
		dAtA[i] = 0x70
		i--
		dAtA[i] = 0xc0
	}
	{
		size, err := m.RecordExample.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		if size > 0 {
			i -= size
			i = encodeVarintProtoVnvn(dAtA, i, uint64(size))
			i--
			dAtA[i] = 0x1
			i--
			dAtA[i] = 0x9a
		}
	}
	i = encodeVarintProtoVnvn(dAtA, i, uint64(999999990))
	i--
	dAtA[i] = 0x1
	i--
	dAtA[i] = 0x80
	return len(dAtA) - i, nil
}

func (m *MessageExample_Head) SetupContext(ctx MessageContext) error {
	return ctx.Message(m, 999999990)
}

const TypeMessageExample_HeadPolymorthID uint64 = 999999990

func (*MessageExample_Head) GetDefaultPolymorphID() uint64 {
	return 999999990
}

func (m *MessageExample_Head) Marshal() (dAtA []byte, err error) {
	size := m.ProtoSize()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MessageExample_Head) MarshalTo(dAtA []byte) (int, error) {
	size := m.ProtoSize()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MessageExample_Head) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l, fieldEnd int
	_, _ = l, fieldEnd
	if m.MsgParam != 0 {
		i = encodeVarintProtoVnvn(dAtA, i, uint64(m.MsgParam))
		i--
		dAtA[i] = 0x70
		i--
		dAtA[i] = 0xc0
	}
	{
		size, err := m.Str.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		if size > 0 {
			i -= size
			i = encodeVarintProtoVnvn(dAtA, i, uint64(size))
			i--
			dAtA[i] = 0x1
			i--
			dAtA[i] = 0xaa
		}
	}
	i = encodeVarintProtoVnvn(dAtA, i, uint64(999999990))
	i--
	dAtA[i] = 0x1
	i--
	dAtA[i] = 0x80
	return len(dAtA) - i, nil
}

func encodeVarintProtoVnvn(dAtA []byte, offset int, v uint64) int {
	offset -= sovProtoVnvn(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}

func init() {
	RegisterMessageType(999999990, "", (*MessageExample)(nil))
	RegisterMessageType(999999990, "Head", (*MessageExample_Head)(nil))
}

func (m *MessageExample) ProtoSize() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if l = m.RecordExample.ProtoSize(); l > 0 {
		n += 2 + l + sovProtoVnvn(uint64(l))
	}
	if m.MsgParam != 0 {
		n += 2 + sovProtoVnvn(uint64(m.MsgParam))
	}
	l = len(m.MsgBytes)
	if l > 0 {
		l++
		n += 2 + l + sovProtoVnvn(uint64(l))
	}
	n += 2 + sovProtoVnvn(999999990)
	return n
}

func (m *MessageExample_Head) ProtoSize() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if l = m.Str.ProtoSize(); l > 0 {
		n += 2 + l + sovProtoVnvn(uint64(l))
	}
	if m.MsgParam != 0 {
		n += 2 + sovProtoVnvn(uint64(m.MsgParam))
	}
	n += 2 + sovProtoVnvn(999999990)
	return n
}

func sovProtoVnvn(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozProtoVnvn(x uint64) (n int) {
	return sovProtoVnvn(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *MessageExample) Unmarshal(dAtA []byte) error {
	return m.UnmarshalWithUnknownCallback(dAtA, skipProtoVnvn)
}
func (m *MessageExample) UnmarshalWithUnknownCallback(dAtA []byte, skipFn func([]byte) (int, error)) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowProtoVnvn
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
			return fmt.Errorf("proto: MessageExample: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MessageExample: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 19:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field RecordExample", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtoVnvn
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
				return ErrInvalidLengthProtoVnvn
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthProtoVnvn
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.RecordExample.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 1800:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field MsgParam", wireType)
			}
			m.MsgParam = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtoVnvn
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.MsgParam |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 1801:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MsgBytes", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtoVnvn
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
				return ErrInvalidLengthProtoVnvn
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthProtoVnvn
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if byteLen > 0 {
				if dAtA[iNdEx] != 132 {
					return ErrExpectedBinaryMarkerProtoVnvn
				}
				iNdEx++
			}
			m.MsgBytes = append(m.MsgBytes[:0], dAtA[iNdEx:postIndex]...)
			if m.MsgBytes == nil {
				m.MsgBytes = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipFn(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				l = iNdEx
				break
			}
			if skippy == 0 {
				if skippy, err = skipProtoVnvn(dAtA[iNdEx:]); err != nil {
					return err
				}
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthProtoVnvn
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
func (m *MessageExample_Head) Unmarshal(dAtA []byte) error {
	return m.UnmarshalWithUnknownCallback(dAtA, skipProtoVnvn)
}
func (m *MessageExample_Head) UnmarshalWithUnknownCallback(dAtA []byte, skipFn func([]byte) (int, error)) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowProtoVnvn
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
			return fmt.Errorf("proto: Head: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Head: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 21:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Str", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtoVnvn
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
				return ErrInvalidLengthProtoVnvn
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthProtoVnvn
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Str.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 1800:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field MsgParam", wireType)
			}
			m.MsgParam = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtoVnvn
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.MsgParam |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipFn(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				l = iNdEx
				break
			}
			if skippy == 0 {
				if skippy, err = skipProtoVnvn(dAtA[iNdEx:]); err != nil {
					return err
				}
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthProtoVnvn
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
func skipProtoVnvn(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowProtoVnvn
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
					return 0, ErrIntOverflowProtoVnvn
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
					return 0, ErrIntOverflowProtoVnvn
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
				return 0, ErrInvalidLengthProtoVnvn
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupProtoVnvn
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthProtoVnvn
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthProtoVnvn        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowProtoVnvn          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupProtoVnvn = fmt.Errorf("proto: unexpected end of group")
	ErrExpectedBinaryMarkerProtoVnvn = fmt.Errorf("proto: binary marker was expected")
)
