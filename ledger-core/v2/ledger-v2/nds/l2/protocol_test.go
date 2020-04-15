// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l2

import (
	"bytes"
	"hash/crc32"
	"io"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

var TestProtocolDescriptor = uniproto.Descriptor{
	SupportedPackets: uniproto.PacketDescriptors{
		0: {Flags: uniproto.NoSourceId | uniproto.OptionalTarget, LengthBits: 16},
	},
}

var _ uniproto.Receiver = &TestProtocolMarshaller{}

type TestProtocolMarshaller struct {
	Count      atomickit.Uint32
	LastFrom   nwapi.Address
	LastPacket uniproto.Packet
	LastBytes  []byte
	LastMsg    string
	LastSigLen int
	LastLarge  bool
	LastError  error
	ReportErr  error
}

func (p *TestProtocolMarshaller) GetPayloadEncrypter() cryptkit.Encrypter {
	panic(throw.Unsupported())
}

func (p *TestProtocolMarshaller) GetPayloadSigner() cryptkit.DataSigner {
	return TestDataSigner{}
}

func (p *TestProtocolMarshaller) PrepareHeader(_ *uniproto.Header, pn pulse.Number) (pulse.Number, error) {
	return pn, nil
}

func (p *TestProtocolMarshaller) VerifyHeader(*uniproto.Header, pulse.Number) error {
	return nil
}

func (p *TestProtocolMarshaller) ReceiveSmallPacket(packet *uniproto.ReceivedPacket, b []byte) {
	p.LastFrom = packet.From
	p.LastPacket = packet.Packet
	p.LastBytes = append([]byte(nil), b...)
	p.LastSigLen = packet.GetSignatureSize()
	p.LastLarge = false
	//p.LastMsg = string(b[packet.GetPayloadOffset() : len(b)-int(p.LastSigLen)])

	p.LastError = packet.NewSmallPayloadDeserializer(b)(func(packet *uniproto.Packet, reader *iokit.LimitedReader) error {
		b := make([]byte, reader.RemainingBytes())
		n, err := io.ReadFull(reader, b)
		p.LastMsg = string(b[:n])
		return err
	})

	p.Count.Add(1)
	return
}

func (p *TestProtocolMarshaller) ReceiveLargePacket(packet *uniproto.ReceivedPacket, preRead []byte, r io.LimitedReader) error {
	p.LastFrom = packet.From
	p.LastPacket = packet.Packet
	p.LastSigLen = packet.GetSignatureSize()
	p.LastLarge = true

	p.LastBytes = nil

	//p.LastBytes = make([]byte, len(preRead)+int(r.N))
	//copy(p.LastBytes, preRead)
	//if _, err := io.ReadFull(&r, p.LastBytes[len(preRead):]); err != nil {
	//	p.LastError = err
	//	return err
	//}
	//p.LastMsg = string(p.LastBytes[packet.GetPayloadOffset() : len(p.LastBytes)-p.LastSigLen])

	p.LastError = packet.NewLargePayloadDeserializer(preRead, r)(func(packet *uniproto.Packet, reader *iokit.LimitedReader) error {
		b := make([]byte, reader.RemainingBytes())
		n, err := io.ReadFull(reader, b)
		p.LastMsg = string(b[:n])
		return err
	})

	p.Count.Add(1)

	return p.ReportErr
}

func (p *TestProtocolMarshaller) SerializeMsg(pt uniproto.ProtocolType, pkt uint8, pn pulse.Number, msg string) []byte {
	var packet uniproto.Packet
	packet.Header.SetProtocolType(pt)
	packet.Header.SetPacketType(pkt)
	packet.Header.SetRelayRestricted(true)
	packet.PulseNumber = pn

	buf := bytes.Buffer{}
	if err := packet.SerializeTo(p, &buf, uint(len(msg)), func(w *iokit.LimitedWriter) error {
		_, err := w.Write([]byte(msg))
		return err
	}); err != nil {
		panic(throw.ErrorWithStack(err))
	}
	return buf.Bytes()
}

func (p *TestProtocolMarshaller) Wait(prevCount uint32) {
	for i := 1000; i > 0; i-- {
		time.Sleep(10 * time.Millisecond)
		if p.Count.Load() > prevCount {
			return
		}
	}
	panic(throw.Impossible())
}

/***********************************/
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

func (v TestVerifierFactory) CreateDataDecrypter(cryptkit.SignatureKey) cryptkit.Decrypter {
	panic("implement me")
}

func (v TestVerifierFactory) CreateDataSigner(cryptkit.SignatureKey) cryptkit.DataSigner {
	panic("implement me")
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
