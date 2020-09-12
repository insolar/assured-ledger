// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rmsbox

import (
	"fmt"
	"io"
	math_bits "math/bits"
)

// This file should not be generated.
//
// message RecordBodyForLazy {
// 	option (insproto.id) = 0;
// 	RecordBody RecordBody = 19;
// }
//

type RecordBodyForLazy struct {
	RecordBody RecordBody `protobuf:"bytes,19,opt,name=RecordBody,proto3" json:"RecordBody"`
}

func (m *RecordBodyForLazy) Reset()         { *m = RecordBodyForLazy{} }

func (m *RecordBodyForLazy) GetRecordBody() RecordBody {
	if m != nil {
		return m.RecordBody
	}
	return RecordBody{}
}

func (m *RecordBodyForLazy) Equal(that interface{}) bool {
	if that == nil {
		return m == nil
	}

	that1, ok := that.(*RecordBodyForLazy)
	if !ok {
		that2, ok := that.(RecordBodyForLazy)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return m == nil
	} else if m == nil {
		return false
	}
	if !m.RecordBody.Equal(&that1.RecordBody) {
		return false
	}
	return true
}

func (m *RecordBodyForLazy) Marshal() (dAtA []byte, err error) {
	size := m.ProtoSize()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	if n != size {
		panic("illegal state")
	}
	return dAtA[:n], nil
}

func (m *RecordBodyForLazy) MarshalTo(dAtA []byte) (int, error) {
	size := m.ProtoSize()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *RecordBodyForLazy) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l, fieldEnd int
	_, _ = l, fieldEnd
	{
		size, err := m.RecordBody.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		if size > 0 {
			i -= size
			i = encodeVarintProtoCatalog(dAtA, i, uint64(size))
			i--
			dAtA[i] = 0x1
			i--
			dAtA[i] = 0x9a
		}
	}
	if i < len(dAtA) {
		i = encodeVarintProtoCatalog(dAtA, i, uint64(0))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0x80
	}
	return len(dAtA) - i, nil
}

func (m *RecordBodyForLazy) ProtoSize() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if l = m.RecordBody.ProtoSize(); l > 0 {
		n += 2 + l + sovProtoCatalog(uint64(l))
	}
	if n > 0 {
		n += 2 + sovProtoCatalog(0)
	}
	return n
}

func (m *RecordBodyForLazy) Unmarshal(dAtA []byte) error {
	return m.UnmarshalWithUnknownCallback(dAtA, skipProtoCatalog)
}
func (m *RecordBodyForLazy) UnmarshalWithUnknownCallback(dAtA []byte, skipFn func([]byte) (int, error)) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowProtoCatalog
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
			return fmt.Errorf("proto: RecordBodyForLazy: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: RecordBodyForLazy: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 19:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field RecordBody", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtoCatalog
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
				return ErrInvalidLengthProtoCatalog
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthProtoCatalog
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.RecordBody.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
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
				if skippy, err = skipProtoCatalog(dAtA[iNdEx:]); err != nil {
					return err
				}
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthProtoCatalog
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

func sovProtoCatalog(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}

func encodeVarintProtoCatalog(dAtA []byte, offset int, v uint64) int {
	offset -= sovProtoCatalog(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}

func skipProtoCatalog(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowProtoCatalog
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
					return 0, ErrIntOverflowProtoCatalog
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
					return 0, ErrIntOverflowProtoCatalog
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
				return 0, ErrInvalidLengthProtoCatalog
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupProtoCatalog
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthProtoCatalog
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthProtoCatalog        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowProtoCatalog          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupProtoCatalog = fmt.Errorf("proto: unexpected end of group")
)
