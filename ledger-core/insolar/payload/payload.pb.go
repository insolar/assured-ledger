// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: insolar/payload/payload.proto

package payload

import (
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	_ "github.com/insolar/assured-ledger/ledger-core/insproto"
	github_com_insolar_assured_ledger_ledger_core_pulse "github.com/insolar/assured-ledger/ledger-core/pulse"
	github_com_insolar_assured_ledger_ledger_core_reference "github.com/insolar/assured-ledger/ledger-core/reference"
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

type Meta struct {
	Payload    []byte                                                         `protobuf:"bytes,20,opt,name=Payload,proto3" json:"Payload"`
	Sender     github_com_insolar_assured_ledger_ledger_core_reference.Global `protobuf:"bytes,21,opt,name=Sender,proto3,customtype=github.com/insolar/assured-ledger/ledger-core/reference.Global" json:"Sender"`
	Receiver   github_com_insolar_assured_ledger_ledger_core_reference.Global `protobuf:"bytes,22,opt,name=Receiver,proto3,customtype=github.com/insolar/assured-ledger/ledger-core/reference.Global" json:"Receiver"`
	Pulse      github_com_insolar_assured_ledger_ledger_core_pulse.Number     `protobuf:"varint,23,opt,name=Pulse,proto3,casttype=github.com/insolar/assured-ledger/ledger-core/pulse.Number" json:"Pulse"`
	ID         []byte                                                         `protobuf:"bytes,24,opt,name=ID,proto3" json:"ID"`
	OriginHash MessageHash                                                    `protobuf:"bytes,25,opt,name=OriginHash,proto3,customtype=MessageHash" json:"OriginHash"`
}

func (m *Meta) Reset()         { *m = Meta{} }
func (m *Meta) String() string { return proto.CompactTextString(m) }
func (*Meta) ProtoMessage()    {}
func (*Meta) Descriptor() ([]byte, []int) {
	return fileDescriptor_33334fec96407f54, []int{0}
}
func (m *Meta) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Meta) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *Meta) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Meta.Merge(m, src)
}
func (m *Meta) XXX_Size() int {
	return m.ProtoSize()
}
func (m *Meta) XXX_DiscardUnknown() {
	xxx_messageInfo_Meta.DiscardUnknown(m)
}

var xxx_messageInfo_Meta proto.InternalMessageInfo

func (m *Meta) GetPayload() []byte {
	if m != nil {
		return m.Payload
	}
	return nil
}

func (m *Meta) GetPulse() github_com_insolar_assured_ledger_ledger_core_pulse.Number {
	if m != nil {
		return m.Pulse
	}
	return 0
}

func (m *Meta) GetID() []byte {
	if m != nil {
		return m.ID
	}
	return nil
}

func init() {
	proto.RegisterType((*Meta)(nil), "payload.Meta")
}

func init() { proto.RegisterFile("insolar/payload/payload.proto", fileDescriptor_33334fec96407f54) }

var fileDescriptor_33334fec96407f54 = []byte{
	// 382 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x92, 0xcd, 0xcc, 0x2b, 0xce,
	0xcf, 0x49, 0x2c, 0xd2, 0x2f, 0x48, 0xac, 0xcc, 0xc9, 0x4f, 0x4c, 0x81, 0xd1, 0x7a, 0x05, 0x45,
	0xf9, 0x25, 0xf9, 0x42, 0xec, 0x50, 0xae, 0x94, 0x6e, 0x7a, 0x66, 0x49, 0x46, 0x69, 0x92, 0x5e,
	0x72, 0x7e, 0xae, 0x7e, 0x7a, 0x7e, 0x7a, 0xbe, 0x3e, 0x58, 0x3e, 0xa9, 0x34, 0x0d, 0xcc, 0x03,
	0x73, 0xc0, 0x2c, 0x88, 0x3e, 0x29, 0x07, 0x24, 0xe5, 0x30, 0x1b, 0x12, 0x8b, 0x8b, 0x4b, 0x8b,
	0x52, 0x53, 0x74, 0x73, 0x52, 0x53, 0xd2, 0x53, 0x8b, 0xf4, 0x21, 0x94, 0x6e, 0x72, 0x7e, 0x51,
	0x2a, 0x48, 0x09, 0xc4, 0x88, 0xcc, 0xbc, 0x62, 0x88, 0x09, 0x4a, 0xcb, 0x98, 0xb9, 0x58, 0x7c,
	0x53, 0x4b, 0x12, 0x85, 0xe4, 0xb8, 0xd8, 0x03, 0x20, 0x8e, 0x90, 0x10, 0x51, 0x60, 0xd4, 0xe0,
	0x71, 0x62, 0x39, 0x71, 0x4f, 0x9e, 0x21, 0x08, 0x26, 0x28, 0x94, 0xc4, 0xc5, 0x16, 0x9c, 0x9a,
	0x97, 0x92, 0x5a, 0x24, 0x21, 0x0a, 0x96, 0xf6, 0x02, 0x49, 0xdf, 0xba, 0x27, 0x6f, 0x47, 0x9a,
	0x13, 0x8a, 0x52, 0xd3, 0x52, 0x8b, 0x52, 0xf3, 0x92, 0x53, 0xf5, 0xdc, 0x73, 0xf2, 0x93, 0x12,
	0x73, 0x3e, 0x4c, 0x53, 0x60, 0x08, 0x82, 0x9a, 0x2c, 0x94, 0xc6, 0xc5, 0x11, 0x94, 0x9a, 0x9c,
	0x9a, 0x59, 0x96, 0x5a, 0x24, 0x21, 0x46, 0x75, 0x5b, 0xe0, 0x66, 0x0b, 0xc5, 0x70, 0xb1, 0x06,
	0x94, 0xe6, 0x14, 0xa7, 0x4a, 0x88, 0x2b, 0x30, 0x6a, 0xf0, 0x3a, 0xb9, 0x81, 0x2c, 0xf9, 0x75,
	0x4f, 0xde, 0x8a, 0x34, 0x4b, 0x0a, 0x40, 0x46, 0xe8, 0xf9, 0x95, 0xe6, 0x26, 0xa5, 0x16, 0x81,
	0x2d, 0x80, 0x18, 0x2a, 0x24, 0xc2, 0xc5, 0xe4, 0xe9, 0x22, 0x21, 0x81, 0x14, 0x88, 0x4c, 0x9e,
	0x2e, 0x42, 0xe6, 0x5c, 0x5c, 0xfe, 0x45, 0x99, 0xe9, 0x99, 0x79, 0x1e, 0x89, 0xc5, 0x19, 0x12,
	0x92, 0x60, 0x59, 0x71, 0xa8, 0xef, 0xb8, 0x7d, 0x53, 0x8b, 0x8b, 0x13, 0xd3, 0x53, 0x41, 0x52,
	0x60, 0x93, 0x90, 0x94, 0x5a, 0xb1, 0x1e, 0xb8, 0x21, 0xff, 0x82, 0xdd, 0xc9, 0xfe, 0xc4, 0x23,
	0x39, 0xc6, 0x0b, 0x8f, 0xe4, 0x18, 0x6f, 0x3c, 0x92, 0x63, 0x7c, 0xf0, 0x48, 0x8e, 0x61, 0xc2,
	0x63, 0x39, 0x86, 0x05, 0x8f, 0xe5, 0x18, 0x2f, 0x3c, 0x96, 0x63, 0xb8, 0xf1, 0x58, 0x8e, 0xe1,
	0xc5, 0x4c, 0x79, 0xc6, 0x0f, 0x33, 0xe5, 0x19, 0x57, 0xcd, 0x92, 0x17, 0x0e, 0x4a, 0x4d, 0xcf,
	0x2c, 0x2e, 0x49, 0x2d, 0x82, 0x9a, 0x1b, 0x52, 0x59, 0x90, 0x9a, 0xc4, 0x06, 0x8e, 0x70, 0x63,
	0x40, 0x00, 0x00, 0x00, 0xff, 0xff, 0xe4, 0x86, 0x31, 0x92, 0x8b, 0x02, 0x00, 0x00,
}

const TypeMetaPolymorthID = 1000

func (*Meta) GetDefaultPolymorphID() uint64 {
	return 1000
}

func (m *Meta) Marshal() (dAtA []byte, err error) {
	size := m.ProtoSize()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Meta) MarshalTo(dAtA []byte) (int, error) {
	size := m.ProtoSize()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Meta) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l, fieldEnd int
	_, _ = l, fieldEnd
	{
		size := m.OriginHash.ProtoSize()
		i -= size
		if _, err := m.OriginHash.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintPayload(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1
	i--
	dAtA[i] = 0xca
	if len(m.ID) > 0 {
		i -= len(m.ID)
		copy(dAtA[i:], m.ID)
		i--
		dAtA[i] = 132
		i = encodeVarintPayload(dAtA, i, uint64(len(m.ID)+1))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0xc2
	}
	if m.Pulse != 0 {
		i = encodeVarintPayload(dAtA, i, uint64(m.Pulse))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0xb8
	}
	{
		size := m.Receiver.ProtoSize()
		i -= size
		if _, err := m.Receiver.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintPayload(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1
	i--
	dAtA[i] = 0xb2
	{
		size := m.Sender.ProtoSize()
		i -= size
		if _, err := m.Sender.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintPayload(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1
	i--
	dAtA[i] = 0xaa
	if len(m.Payload) > 0 {
		i -= len(m.Payload)
		copy(dAtA[i:], m.Payload)
		i--
		dAtA[i] = 132
		i = encodeVarintPayload(dAtA, i, uint64(len(m.Payload)+1))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0xa2
	}
	i = encodeVarintPayload(dAtA, i, uint64(1000))
	i--
	dAtA[i] = 0x1
	i--
	dAtA[i] = 0x80
	return len(dAtA) - i, nil
}

func encodeVarintPayload(dAtA []byte, offset int, v uint64) int {
	offset -= sovPayload(v)
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
	RegisterMessageType(1000, "", (*Meta)(nil))
}

func (m *Meta) ProtoSize() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Payload)
	if l > 0 {
		l++
		n += 2 + l + sovPayload(uint64(l))
	}
	l = m.Sender.ProtoSize()
	n += 2 + l + sovPayload(uint64(l))
	l = m.Receiver.ProtoSize()
	n += 2 + l + sovPayload(uint64(l))
	if m.Pulse != 0 {
		n += 2 + sovPayload(uint64(m.Pulse))
	}
	l = len(m.ID)
	if l > 0 {
		l++
		n += 2 + l + sovPayload(uint64(l))
	}
	l = m.OriginHash.ProtoSize()
	n += 2 + l + sovPayload(uint64(l))
	n += 2 + sovPayload(1000)
	return n
}

func sovPayload(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozPayload(x uint64) (n int) {
	return sovPayload(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Meta) Unmarshal(dAtA []byte) error {
	return m.UnmarshalWithUnknownCallback(dAtA, skipPayload)
}
func (m *Meta) UnmarshalWithUnknownCallback(dAtA []byte, skipFn func([]byte) (int, error)) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowPayload
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
			return fmt.Errorf("proto: Meta: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Meta: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 20:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Payload", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPayload
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
				return ErrInvalidLengthPayload
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthPayload
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if byteLen > 0 {
				if dAtA[iNdEx] != 132 {
					return ErrExpectedBinaryMarkerPayload
				}
				iNdEx++
			}
			m.Payload = append(m.Payload[:0], dAtA[iNdEx:postIndex]...)
			if m.Payload == nil {
				m.Payload = []byte{}
			}
			iNdEx = postIndex
		case 21:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Sender", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPayload
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
				return ErrInvalidLengthPayload
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthPayload
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Sender.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 22:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Receiver", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPayload
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
				return ErrInvalidLengthPayload
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthPayload
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Receiver.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 23:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Pulse", wireType)
			}
			m.Pulse = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPayload
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Pulse |= github_com_insolar_assured_ledger_ledger_core_pulse.Number(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 24:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ID", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPayload
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
				return ErrInvalidLengthPayload
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthPayload
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if byteLen > 0 {
				if dAtA[iNdEx] != 132 {
					return ErrExpectedBinaryMarkerPayload
				}
				iNdEx++
			}
			m.ID = append(m.ID[:0], dAtA[iNdEx:postIndex]...)
			if m.ID == nil {
				m.ID = []byte{}
			}
			iNdEx = postIndex
		case 25:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field OriginHash", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPayload
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
				return ErrInvalidLengthPayload
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthPayload
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.OriginHash.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
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
				if skippy, err = skipPayload(dAtA[iNdEx:]); err != nil {
					return err
				}
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthPayload
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
func skipPayload(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowPayload
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
					return 0, ErrIntOverflowPayload
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
					return 0, ErrIntOverflowPayload
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
				return 0, ErrInvalidLengthPayload
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupPayload
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthPayload
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthPayload        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowPayload          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupPayload = fmt.Errorf("proto: unexpected end of group")
	ErrExpectedBinaryMarkerPayload = fmt.Errorf("proto: binary marker was expected")
)
