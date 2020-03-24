// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package serialization

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPulsarPacketBody_SerializeTo(t *testing.T) {
	b := PulsarPacketBody{}

	buf := bytes.NewBuffer(make([]byte, 0, packetMaxSize))

	err := b.SerializeTo(nil, buf)
	require.NoError(t, err)
	require.Equal(t, 48, buf.Len())
}

func TestPulsarPacketBody_DeserializeFrom(t *testing.T) {
	b1 := PulsarPacketBody{}

	buf := bytes.NewBuffer(make([]byte, 0, packetMaxSize))
	err := b1.SerializeTo(nil, buf)
	require.NoError(t, err)

	b2 := PulsarPacketBody{}
	err = b2.DeserializeFrom(nil, buf)
	require.NoError(t, err)

	require.Equal(t, b1, b2)
}

func TestPulsarPacket_SerializeTo(t *testing.T) {
	p := Packet{
		Header: Header{
			SourceID:   123,
			TargetID:   456,
			ReceiverID: 789,
		},
		EncryptableBody: &PulsarPacketBody{},
	}
	p.Header.setProtocolType(ProtocolTypePulsar)

	buf := bytes.NewBuffer(make([]byte, 0, packetMaxSize))
	s, err := p.SerializeTo(context.Background(), buf, digester, signer)
	require.NoError(t, err)
	require.EqualValues(t, 132, s)

	require.NotEmpty(t, p.PacketSignature)
}

func TestPulsarPacket_DeserializeFrom(t *testing.T) {
	p1 := Packet{
		Header: Header{
			SourceID:   123,
			TargetID:   456,
			ReceiverID: 789,
		},
		EncryptableBody: &PulsarPacketBody{},
	}
	p1.Header.setProtocolType(ProtocolTypePulsar)

	buf := bytes.NewBuffer(make([]byte, 0, packetMaxSize))

	_, err := p1.SerializeTo(context.Background(), buf, digester, signer)
	require.NoError(t, err)

	p2 := Packet{}

	_, err = p2.DeserializeFrom(context.Background(), buf)
	require.NoError(t, err)

	require.Equal(t, p1, p2)
}
