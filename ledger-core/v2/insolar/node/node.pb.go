// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: node.proto

package node

import (
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	github_com_insolar_assured_ledger_ledger_core_v2_reference "github.com/insolar/assured-ledger/ledger-core/v2/reference"
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

type Node struct {
	Polymorph int32                                                             `protobuf:"varint,16,opt,name=polymorph,proto3" json:"polymorph,omitempty"`
	ID        github_com_insolar_assured_ledger_ledger_core_v2_reference.Global `protobuf:"bytes,20,opt,name=ID,proto3,customtype=github.com/insolar/assured-ledger/ledger-core/v2/reference.Global" json:"ID"`
	Role      StaticRole                                                        `protobuf:"varint,21,opt,name=Role,proto3,customtype=StaticRole" json:"Role"`
}

func (m *Node) Reset()      { *m = Node{} }
func (*Node) ProtoMessage() {}
func (*Node) Descriptor() ([]byte, []int) {
	return fileDescriptor_0c843d59d2d938e7, []int{0}
}
func (m *Node) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Node) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Node.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Node) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Node.Merge(m, src)
}
func (m *Node) XXX_Size() int {
	return m.Size()
}
func (m *Node) XXX_DiscardUnknown() {
	xxx_messageInfo_Node.DiscardUnknown(m)
}

var xxx_messageInfo_Node proto.InternalMessageInfo

type NodeList struct {
	Polymorph int32  `protobuf:"varint,16,opt,name=polymorph,proto3" json:"polymorph,omitempty"`
	Nodes     []Node `protobuf:"bytes,20,rep,name=Nodes,proto3" json:"Nodes"`
}

func (m *NodeList) Reset()      { *m = NodeList{} }
func (*NodeList) ProtoMessage() {}
func (*NodeList) Descriptor() ([]byte, []int) {
	return fileDescriptor_0c843d59d2d938e7, []int{1}
}
func (m *NodeList) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *NodeList) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_NodeList.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *NodeList) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NodeList.Merge(m, src)
}
func (m *NodeList) XXX_Size() int {
	return m.Size()
}
func (m *NodeList) XXX_DiscardUnknown() {
	xxx_messageInfo_NodeList.DiscardUnknown(m)
}

var xxx_messageInfo_NodeList proto.InternalMessageInfo

func init() {
	proto.RegisterType((*Node)(nil), "node.Node")
	proto.RegisterType((*NodeList)(nil), "node.NodeList")
}

func init() { proto.RegisterFile("node.proto", fileDescriptor_0c843d59d2d938e7) }

var fileDescriptor_0c843d59d2d938e7 = []byte{
	// 312 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x84, 0x90, 0x3f, 0x4b, 0xc3, 0x40,
	0x18, 0xc6, 0xef, 0x6a, 0x2a, 0x7a, 0x2a, 0x48, 0xa8, 0x10, 0x44, 0xde, 0x86, 0x0e, 0x25, 0x4b,
	0x13, 0xa8, 0x8b, 0xab, 0xa1, 0x20, 0x05, 0x11, 0x89, 0x93, 0x63, 0xfe, 0x5c, 0xd3, 0xc0, 0xb5,
	0x6f, 0xb9, 0xa4, 0x82, 0x9b, 0x1f, 0xc1, 0x6f, 0xa1, 0x1f, 0xa5, 0x63, 0xc7, 0xe2, 0x50, 0xcc,
	0x75, 0x71, 0xec, 0x47, 0x90, 0xbb, 0x0a, 0xba, 0x39, 0xdd, 0xfb, 0x3e, 0xbf, 0xbb, 0xe7, 0x1e,
	0x1e, 0xc6, 0xa6, 0x98, 0x71, 0x7f, 0x26, 0xb1, 0x42, 0xdb, 0xd2, 0xf3, 0x79, 0x2f, 0x2f, 0xaa,
	0xf1, 0x3c, 0xf1, 0x53, 0x9c, 0x04, 0x39, 0xe6, 0x18, 0x18, 0x98, 0xcc, 0x47, 0x66, 0x33, 0x8b,
	0x99, 0x76, 0x8f, 0x3a, 0x6f, 0x94, 0x59, 0x77, 0x98, 0x71, 0xfb, 0x82, 0x1d, 0xce, 0x50, 0x3c,
	0x4f, 0x50, 0xce, 0xc6, 0xce, 0xa9, 0x4b, 0xbd, 0x66, 0xf4, 0x2b, 0xd8, 0x8f, 0xac, 0x31, 0x1c,
	0x38, 0x2d, 0x97, 0x7a, 0xc7, 0xe1, 0x70, 0xb1, 0x6e, 0x93, 0x8f, 0x75, 0xfb, 0xfa, 0xcf, 0x4f,
	0xc5, 0xb4, 0x44, 0x11, 0xcb, 0x20, 0x2e, 0xcb, 0xb9, 0xe4, 0x59, 0x4f, 0xf0, 0x2c, 0xe7, 0x32,
	0xd8, 0x1d, 0xbd, 0x14, 0x25, 0x0f, 0x9e, 0xfa, 0x81, 0xe4, 0x23, 0x2e, 0xf9, 0x34, 0xe5, 0xfe,
	0x8d, 0xc0, 0x24, 0x16, 0x51, 0x63, 0x38, 0xb0, 0xbb, 0xcc, 0x8a, 0x50, 0x70, 0xe7, 0xcc, 0xa5,
	0xde, 0x49, 0x68, 0xff, 0x98, 0xb3, 0x87, 0x2a, 0xae, 0x8a, 0x54, 0x93, 0xc8, 0xf0, 0xce, 0x3d,
	0x3b, 0xd0, 0x41, 0x6f, 0x8b, 0xb2, 0xfa, 0x27, 0x6c, 0x97, 0x35, 0xf5, 0xcd, 0xd2, 0x69, 0xb9,
	0x7b, 0xde, 0x51, 0x9f, 0xf9, 0xa6, 0x24, 0x2d, 0x85, 0x96, 0xb6, 0x8f, 0x76, 0x38, 0xbc, 0x5a,
	0xd4, 0x40, 0x96, 0x35, 0x90, 0x55, 0x0d, 0x64, 0x5b, 0x03, 0x7d, 0x51, 0x40, 0xdf, 0x15, 0xd0,
	0x85, 0x02, 0xba, 0x54, 0x40, 0x3f, 0x15, 0xd0, 0x2f, 0x05, 0x64, 0xab, 0x80, 0xbe, 0x6e, 0x80,
	0x2c, 0x37, 0x40, 0x56, 0x1b, 0x20, 0xc9, 0xbe, 0x29, 0xef, 0xf2, 0x3b, 0x00, 0x00, 0xff, 0xff,
	0x91, 0x2c, 0xed, 0x25, 0x7f, 0x01, 0x00, 0x00,
}

func (this *Node) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*Node)
	if !ok {
		that2, ok := that.(Node)
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
	if !this.Role.Equal(that1.Role) {
		return false
	}
	return true
}
func (this *NodeList) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*NodeList)
	if !ok {
		that2, ok := that.(NodeList)
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
	if len(this.Nodes) != len(that1.Nodes) {
		return false
	}
	for i := range this.Nodes {
		if !this.Nodes[i].Equal(&that1.Nodes[i]) {
			return false
		}
	}
	return true
}
func (this *Node) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 7)
	s = append(s, "&node.Node{")
	s = append(s, "Polymorph: "+fmt.Sprintf("%#v", this.Polymorph)+",\n")
	s = append(s, "ID: "+fmt.Sprintf("%#v", this.ID)+",\n")
	s = append(s, "Role: "+fmt.Sprintf("%#v", this.Role)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func (this *NodeList) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 6)
	s = append(s, "&node.NodeList{")
	s = append(s, "Polymorph: "+fmt.Sprintf("%#v", this.Polymorph)+",\n")
	if this.Nodes != nil {
		vs := make([]Node, len(this.Nodes))
		for i := range vs {
			vs[i] = this.Nodes[i]
		}
		s = append(s, "Nodes: "+fmt.Sprintf("%#v", vs)+",\n")
	}
	s = append(s, "}")
	return strings.Join(s, "")
}
func valueToGoStringNode(v interface{}, typ string) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("func(v %v) *%v { return &v } ( %#v )", typ, typ, pv)
}
func (m *Node) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Node) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Node) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Role != 0 {
		i = encodeVarintNode(dAtA, i, uint64(m.Role))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0xa8
	}
	{
		size := m.ID.Size()
		i -= size
		if _, err := m.ID.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintNode(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1
	i--
	dAtA[i] = 0xa2
	if m.Polymorph != 0 {
		i = encodeVarintNode(dAtA, i, uint64(m.Polymorph))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0x80
	}
	return len(dAtA) - i, nil
}

func (m *NodeList) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *NodeList) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *NodeList) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Nodes) > 0 {
		for iNdEx := len(m.Nodes) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Nodes[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintNode(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x1
			i--
			dAtA[i] = 0xa2
		}
	}
	if m.Polymorph != 0 {
		i = encodeVarintNode(dAtA, i, uint64(m.Polymorph))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0x80
	}
	return len(dAtA) - i, nil
}

func encodeVarintNode(dAtA []byte, offset int, v uint64) int {
	offset -= sovNode(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Node) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Polymorph != 0 {
		n += 2 + sovNode(uint64(m.Polymorph))
	}
	l = m.ID.Size()
	n += 2 + l + sovNode(uint64(l))
	if m.Role != 0 {
		n += 2 + sovNode(uint64(m.Role))
	}
	return n
}

func (m *NodeList) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Polymorph != 0 {
		n += 2 + sovNode(uint64(m.Polymorph))
	}
	if len(m.Nodes) > 0 {
		for _, e := range m.Nodes {
			l = e.Size()
			n += 2 + l + sovNode(uint64(l))
		}
	}
	return n
}

func sovNode(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozNode(x uint64) (n int) {
	return sovNode(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (this *Node) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&Node{`,
		`Polymorph:` + fmt.Sprintf("%v", this.Polymorph) + `,`,
		`ID:` + fmt.Sprintf("%v", this.ID) + `,`,
		`Role:` + fmt.Sprintf("%v", this.Role) + `,`,
		`}`,
	}, "")
	return s
}
func (this *NodeList) String() string {
	if this == nil {
		return "nil"
	}
	repeatedStringForNodes := "[]Node{"
	for _, f := range this.Nodes {
		repeatedStringForNodes += strings.Replace(strings.Replace(f.String(), "Node", "Node", 1), `&`, ``, 1) + ","
	}
	repeatedStringForNodes += "}"
	s := strings.Join([]string{`&NodeList{`,
		`Polymorph:` + fmt.Sprintf("%v", this.Polymorph) + `,`,
		`Nodes:` + repeatedStringForNodes + `,`,
		`}`,
	}, "")
	return s
}
func valueToStringNode(v interface{}) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("*%v", pv)
}
func (m *Node) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowNode
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
			return fmt.Errorf("proto: Node: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Node: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 16:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Polymorph", wireType)
			}
			m.Polymorph = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowNode
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Polymorph |= int32(b&0x7F) << shift
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
					return ErrIntOverflowNode
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
				return ErrInvalidLengthNode
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthNode
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.ID.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 21:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Role", wireType)
			}
			m.Role = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowNode
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Role |= StaticRole(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipNode(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthNode
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthNode
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
func (m *NodeList) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowNode
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
			return fmt.Errorf("proto: NodeList: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: NodeList: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 16:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Polymorph", wireType)
			}
			m.Polymorph = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowNode
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Polymorph |= int32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 20:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Nodes", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowNode
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
				return ErrInvalidLengthNode
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthNode
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Nodes = append(m.Nodes, Node{})
			if err := m.Nodes[len(m.Nodes)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipNode(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthNode
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthNode
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
func skipNode(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowNode
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
					return 0, ErrIntOverflowNode
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
					return 0, ErrIntOverflowNode
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
				return 0, ErrInvalidLengthNode
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupNode
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthNode
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthNode        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowNode          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupNode = fmt.Errorf("proto: unexpected end of group")
)
