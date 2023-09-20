package rms_test

import (
	"crypto/sha256"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

type TestDigester struct {
	altName bool
}

func (p TestDigester) GetDigestMethod() cryptkit.DigestMethod {
	if p.altName {
		return "X-sha224"
	}
	return "sha224"
}

func (p TestDigester) GetDigestSize() int {
	return 224 / 8
}

func (p TestDigester) DigestData(reader io.Reader) cryptkit.Digest {
	h := sha256.New224()
	_, _ = io.Copy(h, reader)
	return cryptkit.DigestOfHash(p, h)
}

func (p TestDigester) DigestBytes(bytes []byte) cryptkit.Digest {
	h := sha256.New224()
	_, _ = h.Write(bytes)
	return cryptkit.DigestOfHash(p, h)
}

func (p TestDigester) NewHasher() cryptkit.DigestHasher {
	return cryptkit.DigestHasher{BasicDigester: p, Hash: sha256.New224()}
}
