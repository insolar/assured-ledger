// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package apinetwork

import (
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
	return v.VerifySignature(hasher.SumToDigest(), b[x:])
}

func (v PacketDataVerifier) VerifySignature(digest cryptkit.Digest, signatureBytes []byte) error {
	signature := cryptkit.NewSignature(longbits.NewMutableFixedSize(signatureBytes), v.Verifier.GetSignatureMethod())

	if !v.Verifier.IsValidDigestSignature(digest, signature) {
		return throw.Violation("packet signature mismatch")
	}
	return nil
}
