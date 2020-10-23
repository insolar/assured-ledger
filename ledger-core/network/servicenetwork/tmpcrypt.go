package servicenetwork

import (
	"hash/crc32"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

const TestSigningMethod cryptkit.SigningMethod = "test-sign"
const TestDigestMethod cryptkit.DigestMethod = "test-hash"
const TestSignatureMethod = cryptkit.SignatureMethod(TestDigestMethod) + "/" + cryptkit.SignatureMethod(TestSigningMethod)
const TestDigestSize = 4

var _ cryptkit.DataSigner = TestDataSigner{}

type TestDataSigner struct{}

func (v TestDataSigner) GetSignatureMethod() cryptkit.SignatureMethod {
	return TestSignatureMethod
}

func (v TestDataSigner) SignDigest(digest cryptkit.Digest) cryptkit.Signature {
	return cryptkit.NewSignature(digest, TestSignatureMethod)
}

func (v TestDataSigner) GetSigningMethod() cryptkit.SigningMethod {
	return TestSigningMethod
}

func (v TestDataSigner) GetDigestMethod() cryptkit.DigestMethod {
	return TestDigestMethod
}

func (v TestDataSigner) GetDigestSize() int {
	return TestDigestSize
}

func (v TestDataSigner) DigestData(r io.Reader) cryptkit.Digest {
	return v.NewHasher().DigestReader(r).SumToDigest()
}

func (v TestDataSigner) DigestBytes(b []byte) cryptkit.Digest {
	return v.NewHasher().DigestBytes(b).SumToDigest()
}

func (v TestDataSigner) NewHasher() cryptkit.DigestHasher {
	return cryptkit.DigestHasher{BasicDigester: v, Hash: crc32.NewIEEE()}
}

/**************************************/
var _ cryptkit.DataSignatureVerifierFactory = TestVerifierFactory{}

type TestVerifierFactory struct{}

func (v TestVerifierFactory) CreateDataEncrypter(key cryptkit.SigningKey) cryptkit.Encrypter {
	panic("implement me")
}

func (v TestVerifierFactory) CreateDataDecrypter(cryptkit.SigningKey) cryptkit.Decrypter {
	panic("implement me")
}

func (v TestVerifierFactory) GetMaxSignatureSize() int {
	return 4
}

func (v TestVerifierFactory) CreateDataSigner(k cryptkit.SigningKey) cryptkit.DataSigner {
	if k.GetSigningMethod() == TestSigningMethod {
		return TestDataSigner{}
	}
	return nil
}

func (v TestVerifierFactory) IsSignatureKeySupported(k cryptkit.SigningKey) bool {
	return k.GetSigningMethod() == TestSigningMethod
}

func (v TestVerifierFactory) CreateDataSignatureVerifier(k cryptkit.SigningKey) cryptkit.DataSignatureVerifier {
	if k.GetSigningMethod() == TestSigningMethod {
		return TestDataVerifier{}
	}
	return nil
}

/**************************************/
var _ cryptkit.DataSignatureVerifier = TestDataVerifier{}

type TestDataVerifier struct {
	TestDataSigner
}

func (t TestDataVerifier) GetDefaultSigningMethod() cryptkit.SigningMethod {
	return TestSigningMethod
}

func (t TestDataVerifier) GetDefaultSignatureMethod() cryptkit.SignatureMethod {
	return TestSignatureMethod
}

func (t TestDataVerifier) IsDigestMethodSupported(m cryptkit.DigestMethod) bool {
	return m == TestDigestMethod
}

func (t TestDataVerifier) IsSigningMethodSupported(m cryptkit.SigningMethod) bool {
	return m == TestSigningMethod
}

func (t TestDataVerifier) IsValidDigestSignature(digest cryptkit.DigestHolder, signature cryptkit.SignatureHolder) bool {
	return longbits.Equal(digest, signature)
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
