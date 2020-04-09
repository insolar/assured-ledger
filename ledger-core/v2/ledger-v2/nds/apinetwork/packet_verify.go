// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package apinetwork

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type PacketDataVerifier struct {
	Verifier cryptkit.DataSignatureVerifier
}

/******************************************************************/

func (v PacketDataVerifier) GetSignatureSize() int {
	return v.Verifier.GetDigestSize()
}

func (v PacketDataVerifier) NewHasher(h *Header) (int, cryptkit.DigestHasher) {
	zeroPrefixLen := h.GetHashingZeroPrefix()
	hasher := v.Verifier.NewHasher()
	_, _ = iokit.WriteZeros(zeroPrefixLen, hasher)
	return zeroPrefixLen, hasher
}

func (v PacketDataVerifier) VerifyWhole(h *Header, b []byte) error {
	skip, hasher := v.NewHasher(h)
	x := len(b) - v.GetSignatureSize()
	if x < 0 {
		return throw.Violation("insufficient length")
	}
	_, _ = hasher.Write(b[skip:x])
	return v.VerifySignature(hasher, b[x:])
}

func (v PacketDataVerifier) VerifySignature(hasher cryptkit.DigestHasher, signatureBytes []byte) error {
	digest := hasher.SumToDigest()
	signature := cryptkit.NewSignature(longbits.NewMutableFixedSize(signatureBytes), v.Verifier.GetSignatureMethod())

	if !v.Verifier.IsValidDigestSignature(digest, signature) {
		return throw.Violation("packet signature mismatch")
	}
	return nil
}

func (v PacketDataVerifier) NewHashingReader(h *Header, preRead []byte, r io.Reader) *cryptkit.HashingTeeReader {
	skip, hasher := v.NewHasher(h)
	tr := cryptkit.NewHashingTeeReader(hasher, r)
	if len(preRead) > skip {
		hasher.DigestBytes(preRead[skip:])
		skip = 0
	} else {
		skip -= len(preRead)
	}
	tr.Skip = skip
	return &tr
}
