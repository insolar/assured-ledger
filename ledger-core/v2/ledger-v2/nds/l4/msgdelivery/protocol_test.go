// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"hash/crc32"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
)

var _ cryptkit.DataSigner = TestDataSigner{}

type TestDataSigner struct{}

func (v TestDataSigner) SignDigest(digest cryptkit.Digest) cryptkit.Signature {
	return cryptkit.NewSignature(digest, testSignatureMethod)
}

func (v TestDataSigner) GetSigningMethod() cryptkit.SigningMethod {
	return testSigningMethod
}

const testSigningMethod = "test-sign"
const testDigestMethod = "test-hash"
const testSignatureMethod = testDigestMethod + "/" + testSigningMethod
const testDigestSize = 4

func (v TestDataSigner) GetDigestMethod() cryptkit.DigestMethod {
	return testDigestMethod
}

func (v TestDataSigner) GetDigestSize() int {
	return testDigestSize
}

func (v TestDataSigner) DigestData(r io.Reader) cryptkit.Digest {
	return v.NewHasher().DigestReader(r).SumToDigest()
}

func (v TestDataSigner) DigestBytes(b []byte) cryptkit.Digest {
	return v.NewHasher().DigestBytes(b).SumToDigest()
}

func (v TestDataSigner) NewHasher() cryptkit.DigestHasher {
	return cryptkit.DigestHasher{v, crc32.NewIEEE()}
}

/**************************************/
var _ cryptkit.DataSignatureVerifierFactory = TestVerifierFactory{}

type TestVerifierFactory struct{}

func (v TestVerifierFactory) CreateDataEncrypter(key cryptkit.SignatureKey) cryptkit.Encrypter {
	panic("implement me")
}

func (v TestVerifierFactory) CreateDataDecrypter(cryptkit.SignatureKey) cryptkit.Decrypter {
	panic("implement me")
}

func (v TestVerifierFactory) GetMaxSignatureSize() int {
	return 4
}

func (v TestVerifierFactory) CreateDataSigner(k cryptkit.SignatureKey) cryptkit.DataSigner {
	if k.GetSigningMethod() == testSigningMethod {
		return TestDataSigner{}
	}
	return nil
}

func (v TestVerifierFactory) IsSignatureKeySupported(k cryptkit.SignatureKey) bool {
	return k.GetSigningMethod() == testSigningMethod
}

func (v TestVerifierFactory) CreateDataSignatureVerifier(k cryptkit.SignatureKey) cryptkit.DataSignatureVerifier {
	if k.GetSigningMethod() == testSigningMethod {
		return TestDataVerifier{}
	}
	return nil
}

/**************************************/
var _ cryptkit.DataSignatureVerifier = TestDataVerifier{}

type TestDataVerifier struct {
	TestDataSigner
}

func (t TestDataVerifier) GetSignatureMethod() cryptkit.SignatureMethod {
	return testSignatureMethod
}

func (t TestDataVerifier) IsDigestMethodSupported(m cryptkit.DigestMethod) bool {
	return m == testDigestMethod
}

func (t TestDataVerifier) IsSignMethodSupported(m cryptkit.SigningMethod) bool {
	return m == testSigningMethod
}

func (t TestDataVerifier) IsSignOfSignatureMethodSupported(m cryptkit.SignatureMethod) bool {
	return m.SignMethod() == testSigningMethod
}

func (t TestDataVerifier) IsValidDigestSignature(digest cryptkit.DigestHolder, signature cryptkit.SignatureHolder) bool {
	return longbits.EqualFixedLenWriterTo(digest, signature)
}

func (t TestDataVerifier) IsValidDataSignature(data io.Reader, signature cryptkit.SignatureHolder) bool {
	digest := t.NewHasher().DigestReader(data).SumToDigest()
	return t.IsValidDigestSignature(digest, signature)
}

/**************************************/
var _ nwapi.Serializable = &TestString{}

type TestString struct {
	S string
}

func (v *TestString) ByteSize() uint {
	return uint(len(v.S))
}

func (v *TestString) SerializeTo(_ nwapi.SerializationContext, writer *iokit.LimitedWriter) error {
	_, err := writer.Write([]byte(v.S))
	return err
}

func (v *TestString) DeserializeFrom(_ nwapi.DeserializationContext, reader *iokit.LimitedReader) error {
	b := make([]byte, reader.RemainingBytes())
	if _, err := io.ReadFull(reader, b); err != nil {
		return err
	}
	v.S = string(b)
	return nil
}

func (v TestString) String() string {
	return v.S
}

/**************************************/
var _ nwapi.DeserializationFactory = TestDeserializationFactory{}

type TestDeserializationFactory struct{}

func (TestDeserializationFactory) DeserializePayloadFrom(ctx nwapi.DeserializationContext, _ nwapi.PayloadCompleteness, reader *iokit.LimitedReader) (nwapi.Serializable, error) {
	var s TestString
	if err := s.DeserializeFrom(ctx, reader); err != nil {
		return nil, err
	}
	return &s, nil
}
