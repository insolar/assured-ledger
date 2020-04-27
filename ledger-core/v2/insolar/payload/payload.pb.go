// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: payload.proto

package payload

import (
	bytes "bytes"
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	github_com_insolar_assured_ledger_ledger_core_v2_insolar "github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	io "io"
	math "math"
	math_bits "math/bits"
	reflect "reflect"
	strings "strings"
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
	Polymorph  uint32                                                               `protobuf:"varint,16,opt,name=Polymorph,proto3" json:"Polymorph,omitempty"`
	Payload    []byte                                                               `protobuf:"bytes,20,opt,name=Payload,proto3" json:"Payload,omitempty"`
	Sender     github_com_insolar_assured_ledger_ledger_core_v2_insolar.Reference   `protobuf:"bytes,21,opt,name=Sender,proto3,customtype=github.com/insolar/assured-ledger/ledger-core/v2/insolar.Reference" json:"Sender"`
	Receiver   github_com_insolar_assured_ledger_ledger_core_v2_insolar.Reference   `protobuf:"bytes,22,opt,name=Receiver,proto3,customtype=github.com/insolar/assured-ledger/ledger-core/v2/insolar.Reference" json:"Receiver"`
	Pulse      github_com_insolar_assured_ledger_ledger_core_v2_insolar.PulseNumber `protobuf:"bytes,23,opt,name=Pulse,proto3,customtype=github.com/insolar/assured-ledger/ledger-core/v2/insolar.PulseNumber" json:"Pulse"`
	ID         []byte                                                               `protobuf:"bytes,24,opt,name=ID,proto3" json:"ID,omitempty"`
	OriginHash MessageHash                                                          `protobuf:"bytes,25,opt,name=OriginHash,proto3,customtype=MessageHash" json:"OriginHash"`
}

func (m *Meta) Reset()      { *m = Meta{} }
func (*Meta) ProtoMessage() {}
func (*Meta) Descriptor() ([]byte, []int) {
	return fileDescriptor_678c914f1bee6d56, []int{0}
}
func (m *Meta) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Meta) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Meta.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Meta) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Meta.Merge(m, src)
}
func (m *Meta) XXX_Size() int {
	return m.Size()
}
func (m *Meta) XXX_DiscardUnknown() {
	xxx_messageInfo_Meta.DiscardUnknown(m)
}

var xxx_messageInfo_Meta proto.InternalMessageInfo

func (m *Meta) GetPolymorph() uint32 {
	if m != nil {
		return m.Polymorph
	}
	return 0
}

func (m *Meta) GetPayload() []byte {
	if m != nil {
		return m.Payload
	}
	return nil
}

func (m *Meta) GetID() []byte {
	if m != nil {
		return m.ID
	}
	return nil
}

type Error struct {
	Polymorph uint32    `protobuf:"varint,16,opt,name=Polymorph,proto3" json:"Polymorph,omitempty"`
	Code      ErrorCode `protobuf:"varint,20,opt,name=Code,proto3,customtype=ErrorCode" json:"Code"`
	Text      string    `protobuf:"bytes,21,opt,name=Text,proto3" json:"Text,omitempty"`
}

func (m *Error) Reset()      { *m = Error{} }
func (*Error) ProtoMessage() {}
func (*Error) Descriptor() ([]byte, []int) {
	return fileDescriptor_678c914f1bee6d56, []int{1}
}
func (m *Error) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Error) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Error.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Error) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Error.Merge(m, src)
}
func (m *Error) XXX_Size() int {
	return m.Size()
}
func (m *Error) XXX_DiscardUnknown() {
	xxx_messageInfo_Error.DiscardUnknown(m)
}

var xxx_messageInfo_Error proto.InternalMessageInfo

func (m *Error) GetPolymorph() uint32 {
	if m != nil {
		return m.Polymorph
	}
	return 0
}

func (m *Error) GetText() string {
	if m != nil {
		return m.Text
	}
	return ""
}

type ID struct {
	Polymorph uint32                                                      `protobuf:"varint,16,opt,name=Polymorph,proto3" json:"Polymorph,omitempty"`
	ID        github_com_insolar_assured_ledger_ledger_core_v2_insolar.ID `protobuf:"bytes,20,opt,name=ID,proto3,customtype=github.com/insolar/assured-ledger/ledger-core/v2/insolar.ID" json:"ID"`
}

func (m *ID) Reset()      { *m = ID{} }
func (*ID) ProtoMessage() {}
func (*ID) Descriptor() ([]byte, []int) {
	return fileDescriptor_678c914f1bee6d56, []int{2}
}
func (m *ID) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ID) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ID.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ID) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ID.Merge(m, src)
}
func (m *ID) XXX_Size() int {
	return m.Size()
}
func (m *ID) XXX_DiscardUnknown() {
	xxx_messageInfo_ID.DiscardUnknown(m)
}

var xxx_messageInfo_ID proto.InternalMessageInfo

func (m *ID) GetPolymorph() uint32 {
	if m != nil {
		return m.Polymorph
	}
	return 0
}

type IDs struct {
	Polymorph uint32                                                        `protobuf:"varint,16,opt,name=Polymorph,proto3" json:"Polymorph,omitempty"`
	IDs       []github_com_insolar_assured_ledger_ledger_core_v2_insolar.ID `protobuf:"bytes,20,rep,name=IDs,proto3,customtype=github.com/insolar/assured-ledger/ledger-core/v2/insolar.ID" json:"IDs"`
}

func (m *IDs) Reset()      { *m = IDs{} }
func (*IDs) ProtoMessage() {}
func (*IDs) Descriptor() ([]byte, []int) {
	return fileDescriptor_678c914f1bee6d56, []int{3}
}
func (m *IDs) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *IDs) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_IDs.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *IDs) XXX_Merge(src proto.Message) {
	xxx_messageInfo_IDs.Merge(m, src)
}
func (m *IDs) XXX_Size() int {
	return m.Size()
}
func (m *IDs) XXX_DiscardUnknown() {
	xxx_messageInfo_IDs.DiscardUnknown(m)
}

var xxx_messageInfo_IDs proto.InternalMessageInfo

func (m *IDs) GetPolymorph() uint32 {
	if m != nil {
		return m.Polymorph
	}
	return 0
}

func init() {
	proto.RegisterType((*Meta)(nil), "payload.Meta")
	proto.RegisterType((*Error)(nil), "payload.Error")
	proto.RegisterType((*ID)(nil), "payload.ID")
	proto.RegisterType((*IDs)(nil), "payload.IDs")
}

func init() { proto.RegisterFile("payload.proto", fileDescriptor_678c914f1bee6d56) }

var fileDescriptor_678c914f1bee6d56 = []byte{
	// 438 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xac, 0x93, 0xcf, 0x6e, 0xd3, 0x40,
	0x10, 0xc6, 0xbd, 0x4d, 0xda, 0x92, 0x85, 0x20, 0x58, 0x0a, 0x2c, 0x08, 0x6d, 0xa3, 0x48, 0x48,
	0xbd, 0x24, 0x96, 0x28, 0x37, 0x6e, 0xae, 0x91, 0x30, 0xa2, 0x10, 0x6d, 0xe1, 0x8e, 0xff, 0x4c,
	0x1c, 0x4b, 0x8e, 0x37, 0xda, 0xb5, 0x0b, 0xe5, 0xc4, 0x23, 0xf0, 0x18, 0xdc, 0x79, 0x89, 0x1e,
	0x73, 0xac, 0x38, 0x54, 0xc4, 0xb9, 0x70, 0xec, 0x23, 0x20, 0x8f, 0x5d, 0xda, 0x5b, 0xa4, 0x2a,
	0x27, 0xef, 0xe7, 0x6f, 0xf6, 0xfb, 0xad, 0x67, 0xbc, 0xb4, 0x3b, 0xf3, 0x4f, 0x52, 0xe5, 0x47,
	0xc3, 0x99, 0x56, 0xb9, 0x62, 0xdb, 0x8d, 0x7c, 0x3a, 0x88, 0x93, 0x7c, 0x52, 0x04, 0xc3, 0x50,
	0x4d, 0xed, 0x58, 0xc5, 0xca, 0x46, 0x3f, 0x28, 0xc6, 0xa8, 0x50, 0xe0, 0xaa, 0xde, 0xd7, 0xff,
	0xd5, 0xa2, 0xed, 0x43, 0xc8, 0x7d, 0xf6, 0x8c, 0x76, 0x46, 0x2a, 0x3d, 0x99, 0x2a, 0x3d, 0x9b,
	0xf0, 0x7b, 0x3d, 0xb2, 0xd7, 0x95, 0x57, 0x2f, 0x18, 0xa7, 0xdb, 0xa3, 0x1a, 0xc0, 0x77, 0x7a,
	0x64, 0xef, 0x8e, 0xbc, 0x94, 0x2c, 0xa0, 0x5b, 0x47, 0x90, 0x45, 0xa0, 0xf9, 0xc3, 0xca, 0x70,
	0xde, 0x9e, 0x9e, 0xef, 0x5a, 0xbf, 0xcf, 0x77, 0x9d, 0x6b, 0xe7, 0x48, 0x32, 0xa3, 0x52, 0x5f,
	0xdb, 0xbe, 0x31, 0x85, 0x86, 0x68, 0x90, 0x42, 0x14, 0x83, 0xb6, 0xeb, 0xc7, 0x20, 0x54, 0x1a,
	0xec, 0xe3, 0x17, 0x97, 0x55, 0x43, 0x09, 0x63, 0xd0, 0x90, 0x85, 0x20, 0x9b, 0x64, 0x36, 0xa6,
	0xb7, 0x24, 0x84, 0x90, 0x1c, 0x83, 0xe6, 0x8f, 0xd6, 0x4e, 0xf9, 0x9f, 0xcd, 0x02, 0xba, 0x39,
	0x2a, 0x52, 0x03, 0xfc, 0x31, 0x42, 0xde, 0x35, 0x10, 0xf7, 0xc6, 0x10, 0x4c, 0x7b, 0x5f, 0x4c,
	0x03, 0xd0, 0xb2, 0x8e, 0x66, 0x77, 0xe9, 0x86, 0xe7, 0x72, 0x8e, 0x4d, 0xdc, 0xf0, 0x5c, 0xb6,
	0x4f, 0xe9, 0x07, 0x9d, 0xc4, 0x49, 0xf6, 0xc6, 0x37, 0x13, 0xfe, 0x04, 0xc1, 0x0f, 0x1a, 0xf0,
	0xed, 0x43, 0x30, 0xc6, 0x8f, 0xa1, 0xb2, 0xe4, 0xb5, 0xb2, 0xfe, 0x67, 0xba, 0xf9, 0x5a, 0x6b,
	0xa5, 0x57, 0x4c, 0xed, 0x39, 0x6d, 0x1f, 0xa8, 0x08, 0x70, 0x64, 0x5d, 0xe7, 0x7e, 0x93, 0xda,
	0xc1, 0xad, 0x95, 0x21, 0xd1, 0x66, 0x8c, 0xb6, 0x3f, 0xc2, 0xd7, 0x1c, 0x07, 0xd8, 0x91, 0xb8,
	0xee, 0x7f, 0xa9, 0x8e, 0xb9, 0x22, 0xfe, 0x08, 0x3f, 0x05, 0xff, 0x07, 0xe7, 0xa0, 0x09, 0x7f,
	0x75, 0xe3, 0x5e, 0x79, 0x6e, 0xd5, 0x8f, 0xfe, 0x37, 0xda, 0xf2, 0x5c, 0xb3, 0x82, 0xfc, 0x09,
	0x8b, 0xf8, 0x4e, 0xaf, 0xb5, 0x2e, 0x74, 0x95, 0xe7, 0xbc, 0x9c, 0x2f, 0x84, 0x75, 0xb6, 0x10,
	0xd6, 0xc5, 0x42, 0x90, 0xef, 0xa5, 0x20, 0x3f, 0x4b, 0x41, 0x4e, 0x4b, 0x41, 0xe6, 0xa5, 0x20,
	0x7f, 0x4a, 0x41, 0xfe, 0x96, 0xc2, 0xba, 0x28, 0x05, 0xf9, 0xb1, 0x14, 0xd6, 0x7c, 0x29, 0xac,
	0xb3, 0xa5, 0xb0, 0x82, 0x2d, 0xbc, 0x49, 0xfb, 0xff, 0x02, 0x00, 0x00, 0xff, 0xff, 0xf6, 0x0b,
	0xcc, 0x4b, 0x92, 0x03, 0x00, 0x00,
}

func (this *Meta) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*Meta)
	if !ok {
		that2, ok := that.(Meta)
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
	if this.Polymorph != that1.Polymorph {
		return false
	}
	if !bytes.Equal(this.Payload, that1.Payload) {
		return false
	}
	if !this.Sender.Equal(that1.Sender) {
		return false
	}
	if !this.Receiver.Equal(that1.Receiver) {
		return false
	}
	if !this.Pulse.Equal(that1.Pulse) {
		return false
	}
	if !bytes.Equal(this.ID, that1.ID) {
		return false
	}
	if !this.OriginHash.Equal(that1.OriginHash) {
		return false
	}
	return true
}
func (this *Error) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*Error)
	if !ok {
		that2, ok := that.(Error)
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
	if this.Polymorph != that1.Polymorph {
		return false
	}
	if !this.Code.Equal(that1.Code) {
		return false
	}
	if this.Text != that1.Text {
		return false
	}
	return true
}
func (this *ID) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*ID)
	if !ok {
		that2, ok := that.(ID)
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
	if this.Polymorph != that1.Polymorph {
		return false
	}
	if !this.ID.Equal(that1.ID) {
		return false
	}
	return true
}
func (this *IDs) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*IDs)
	if !ok {
		that2, ok := that.(IDs)
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
	if this.Polymorph != that1.Polymorph {
		return false
	}
	if len(this.IDs) != len(that1.IDs) {
		return false
	}
	for i := range this.IDs {
		if !this.IDs[i].Equal(that1.IDs[i]) {
			return false
		}
	}
	return true
}
func (this *Meta) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 11)
	s = append(s, "&payload.Meta{")
	s = append(s, "Polymorph: "+fmt.Sprintf("%#v", this.Polymorph)+",\n")
	s = append(s, "Payload: "+fmt.Sprintf("%#v", this.Payload)+",\n")
	s = append(s, "Sender: "+fmt.Sprintf("%#v", this.Sender)+",\n")
	s = append(s, "Receiver: "+fmt.Sprintf("%#v", this.Receiver)+",\n")
	s = append(s, "Pulse: "+fmt.Sprintf("%#v", this.Pulse)+",\n")
	s = append(s, "ID: "+fmt.Sprintf("%#v", this.ID)+",\n")
	s = append(s, "OriginHash: "+fmt.Sprintf("%#v", this.OriginHash)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func (this *Error) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 7)
	s = append(s, "&payload.Error{")
	s = append(s, "Polymorph: "+fmt.Sprintf("%#v", this.Polymorph)+",\n")
	s = append(s, "Code: "+fmt.Sprintf("%#v", this.Code)+",\n")
	s = append(s, "Text: "+fmt.Sprintf("%#v", this.Text)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func (this *ID) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 6)
	s = append(s, "&payload.ID{")
	s = append(s, "Polymorph: "+fmt.Sprintf("%#v", this.Polymorph)+",\n")
	s = append(s, "ID: "+fmt.Sprintf("%#v", this.ID)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func (this *IDs) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 6)
	s = append(s, "&payload.IDs{")
	s = append(s, "Polymorph: "+fmt.Sprintf("%#v", this.Polymorph)+",\n")
	s = append(s, "IDs: "+fmt.Sprintf("%#v", this.IDs)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func valueToGoStringPayload(v interface{}, typ string) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("func(v %v) *%v { return &v } ( %#v )", typ, typ, pv)
}
func (m *Meta) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Meta) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Meta) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size := m.OriginHash.Size()
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
		i = encodeVarintPayload(dAtA, i, uint64(len(m.ID)))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0xc2
	}
	{
		size := m.Pulse.Size()
		i -= size
		if _, err := m.Pulse.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintPayload(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1
	i--
	dAtA[i] = 0xba
	{
		size := m.Receiver.Size()
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
		size := m.Sender.Size()
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
		i = encodeVarintPayload(dAtA, i, uint64(len(m.Payload)))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0xa2
	}
	if m.Polymorph != 0 {
		i = encodeVarintPayload(dAtA, i, uint64(m.Polymorph))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0x80
	}
	return len(dAtA) - i, nil
}

func (m *Error) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Error) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Error) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Text) > 0 {
		i -= len(m.Text)
		copy(dAtA[i:], m.Text)
		i = encodeVarintPayload(dAtA, i, uint64(len(m.Text)))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0xaa
	}
	if m.Code != 0 {
		i = encodeVarintPayload(dAtA, i, uint64(m.Code))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0xa0
	}
	if m.Polymorph != 0 {
		i = encodeVarintPayload(dAtA, i, uint64(m.Polymorph))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0x80
	}
	return len(dAtA) - i, nil
}

func (m *ID) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ID) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ID) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size := m.ID.Size()
		i -= size
		if _, err := m.ID.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintPayload(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1
	i--
	dAtA[i] = 0xa2
	if m.Polymorph != 0 {
		i = encodeVarintPayload(dAtA, i, uint64(m.Polymorph))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0x80
	}
	return len(dAtA) - i, nil
}

func (m *IDs) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *IDs) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *IDs) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.IDs) > 0 {
		for iNdEx := len(m.IDs) - 1; iNdEx >= 0; iNdEx-- {
			{
				size := m.IDs[iNdEx].Size()
				i -= size
				if _, err := m.IDs[iNdEx].MarshalTo(dAtA[i:]); err != nil {
					return 0, err
				}
				i = encodeVarintPayload(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x1
			i--
			dAtA[i] = 0xa2
		}
	}
	if m.Polymorph != 0 {
		i = encodeVarintPayload(dAtA, i, uint64(m.Polymorph))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0x80
	}
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
func (m *Meta) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Polymorph != 0 {
		n += 2 + sovPayload(uint64(m.Polymorph))
	}
	l = len(m.Payload)
	if l > 0 {
		n += 2 + l + sovPayload(uint64(l))
	}
	l = m.Sender.Size()
	n += 2 + l + sovPayload(uint64(l))
	l = m.Receiver.Size()
	n += 2 + l + sovPayload(uint64(l))
	l = m.Pulse.Size()
	n += 2 + l + sovPayload(uint64(l))
	l = len(m.ID)
	if l > 0 {
		n += 2 + l + sovPayload(uint64(l))
	}
	l = m.OriginHash.Size()
	n += 2 + l + sovPayload(uint64(l))
	return n
}

func (m *Error) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Polymorph != 0 {
		n += 2 + sovPayload(uint64(m.Polymorph))
	}
	if m.Code != 0 {
		n += 2 + sovPayload(uint64(m.Code))
	}
	l = len(m.Text)
	if l > 0 {
		n += 2 + l + sovPayload(uint64(l))
	}
	return n
}

func (m *ID) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Polymorph != 0 {
		n += 2 + sovPayload(uint64(m.Polymorph))
	}
	l = m.ID.Size()
	n += 2 + l + sovPayload(uint64(l))
	return n
}

func (m *IDs) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Polymorph != 0 {
		n += 2 + sovPayload(uint64(m.Polymorph))
	}
	if len(m.IDs) > 0 {
		for _, e := range m.IDs {
			l = e.Size()
			n += 2 + l + sovPayload(uint64(l))
		}
	}
	return n
}

func sovPayload(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozPayload(x uint64) (n int) {
	return sovPayload(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (this *Meta) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&Meta{`,
		`Polymorph:` + fmt.Sprintf("%v", this.Polymorph) + `,`,
		`Payload:` + fmt.Sprintf("%v", this.Payload) + `,`,
		`Sender:` + fmt.Sprintf("%v", this.Sender) + `,`,
		`Receiver:` + fmt.Sprintf("%v", this.Receiver) + `,`,
		`Pulse:` + fmt.Sprintf("%v", this.Pulse) + `,`,
		`ID:` + fmt.Sprintf("%v", this.ID) + `,`,
		`OriginHash:` + fmt.Sprintf("%v", this.OriginHash) + `,`,
		`}`,
	}, "")
	return s
}
func (this *Error) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&Error{`,
		`Polymorph:` + fmt.Sprintf("%v", this.Polymorph) + `,`,
		`Code:` + fmt.Sprintf("%v", this.Code) + `,`,
		`Text:` + fmt.Sprintf("%v", this.Text) + `,`,
		`}`,
	}, "")
	return s
}
func (this *ID) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&ID{`,
		`Polymorph:` + fmt.Sprintf("%v", this.Polymorph) + `,`,
		`ID:` + fmt.Sprintf("%v", this.ID) + `,`,
		`}`,
	}, "")
	return s
}
func (this *IDs) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&IDs{`,
		`Polymorph:` + fmt.Sprintf("%v", this.Polymorph) + `,`,
		`IDs:` + fmt.Sprintf("%v", this.IDs) + `,`,
		`}`,
	}, "")
	return s
}
func valueToStringPayload(v interface{}) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("*%v", pv)
}
func (m *Meta) Unmarshal(dAtA []byte) error {
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
		case 16:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Polymorph", wireType)
			}
			m.Polymorph = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPayload
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Polymorph |= uint32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
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
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Pulse", wireType)
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
			if err := m.Pulse.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
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
			skippy, err := skipPayload(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthPayload
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
func (m *Error) Unmarshal(dAtA []byte) error {
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
			return fmt.Errorf("proto: Error: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Error: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 16:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Polymorph", wireType)
			}
			m.Polymorph = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPayload
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Polymorph |= uint32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 20:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Code", wireType)
			}
			m.Code = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPayload
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Code |= ErrorCode(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 21:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Text", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPayload
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthPayload
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPayload
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Text = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipPayload(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthPayload
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
func (m *ID) Unmarshal(dAtA []byte) error {
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
			return fmt.Errorf("proto: ID: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ID: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 16:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Polymorph", wireType)
			}
			m.Polymorph = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPayload
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Polymorph |= uint32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 20:
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
			if err := m.ID.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipPayload(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthPayload
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
func (m *IDs) Unmarshal(dAtA []byte) error {
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
			return fmt.Errorf("proto: IDs: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: IDs: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 16:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Polymorph", wireType)
			}
			m.Polymorph = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPayload
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Polymorph |= uint32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 20:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field IDs", wireType)
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
			var v github_com_insolar_assured_ledger_ledger_core_v2_insolar.ID
			m.IDs = append(m.IDs, v)
			if err := m.IDs[len(m.IDs)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipPayload(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthPayload
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
)
