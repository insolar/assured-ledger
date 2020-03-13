/*
 * Copyright 2020 Insolar Network Ltd.
 * All rights reserved.
 * This material is licensed under the Insolar License version 1.0,
 * available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package rms

import (
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
)

var pcp PlatformCryptographyProvider = testPlatformCryptographyProvider{}

type testPlatformCryptographyProvider struct{}

func (t testPlatformCryptographyProvider) PlatformCryptographyProvider() {
}

func (t testPlatformCryptographyProvider) GetPlatformCryptographyScheme() CryptographyScheme {
	return testCryptographyScheme{}
}

func (t testPlatformCryptographyProvider) GetExtensionCryptographyScheme(ExtensionId) CryptographyScheme {
	return testCryptographyScheme{}
}

type testCryptographyScheme struct{}

func (t testCryptographyScheme) CryptographyScheme() {}

func (t testCryptographyScheme) GetRecordBodyDigester() cryptkit.DataDigester {
	return typeDataDigester{}
}

type typeDataDigester struct {
}

func (t typeDataDigester) GetDigestMethod() cryptkit.DigestMethod {
	return "test"
}

func (t typeDataDigester) GetDigestSize() int {
	return 8
}

func (t typeDataDigester) DigestData(io.Reader) cryptkit.Digest {
	panic("implement me")
}

func (t typeDataDigester) DigestBytes(a []byte) cryptkit.Digest {
	b := make([]byte, 8)
	copy(b, a)
	return cryptkit.NewDigest(longbits.NewMutableFixedSize(b), "test")
}

type testPayloadProvider struct {
	data []byte
}

func (v *testPayloadProvider) ProtoSize() int {
	return len(v.data)
}

func (v *testPayloadProvider) MarshalTo(dAtA []byte) (int, error) {
	return copy(dAtA, v.data), nil
}

func (v *testPayloadProvider) Unmarshal(b []byte) error {
	v.data = b
	return nil
}

func (v *testPayloadProvider) GetPayloadContainer(CryptographyScheme) GoGoMarshaller {
	return v
}

func TestMsgSerialize(t *testing.T) {
	msg1 := MsgExample{}
	msg1.MsgParam = 11
	msg1.MsgPolymorph = 1
	msg1.Polymorph = 2
	msg1.Str = "123"
	msg1.MsgBytes = []byte{1}
	msg1.Ref1 = []byte{10, 11, 12}
	msg1.Body().BodyPayload = &testPayloadProvider{[]byte("payload")}
	msg1.Body().Extensions = []ExtensionProvider{{&testPayloadProvider{[]byte("ext1")}, 1}}

	nvlp1 := NewMessageEnvelope(pcp, &msg1)
	b, err := nvlp1.Marshal()
	require.NoError(t, err)

	fmt.Printf("%+x\n", b)

	msg2 := MsgExample{}
	msg2.Body().BodyPayload = &testPayloadProvider{[]byte("")}
	msg2.Body().Extensions = []ExtensionProvider{{&testPayloadProvider{[]byte("")}, 1}}

	nvlp2 := NewMessageEnvelope(pcp, &msg2)
	err = nvlp2.Unmarshal(b)
	require.NoError(t, err)

	require.Equal(t, msg1, msg2)
}
