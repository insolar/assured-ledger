// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package testutils

import (
	"crypto"
	"hash"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"

	uuid "github.com/satori/go.uuid"
	"golang.org/x/crypto/sha3"

	"github.com/insolar/assured-ledger/ledger-core/v2/cryptography"
)

const letterBytes = "abcdef0123456789"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

func RandomHashWithLength(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			sb.WriteByte(letterBytes[idx])
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return sb.String()
}

func RandomEthHash() string {
	return "0x" + RandomHashWithLength(64)
}

func RandomEthMigrationAddress() string {
	return "0x" + RandomHashWithLength(40)
}

// RandomString generates random uuid and return it as a string.
func RandomString() string {
	newUUID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	return newUUID.String()
}

type cryptographySchemeMock struct{}
type hasherMock struct {
	h hash.Hash
}

func (m *hasherMock) Write(p []byte) (n int, err error) {
	return m.h.Write(p)
}

func (m *hasherMock) Sum(b []byte) []byte {
	return m.h.Sum(b)
}

func (m *hasherMock) Reset() {
	m.h.Reset()
}

func (m *hasherMock) Size() int {
	return m.h.Size()
}

func (m *hasherMock) BlockSize() int {
	return m.h.BlockSize()
}

func (m *hasherMock) Hash(val []byte) []byte {
	_, _ = m.h.Write(val)
	return m.h.Sum(nil)
}

func (m *cryptographySchemeMock) ReferenceHasher() cryptography.Hasher {
	return &hasherMock{h: sha3.New512()}
}

func (m *cryptographySchemeMock) IntegrityHasher() cryptography.Hasher {
	return &hasherMock{h: sha3.New512()}
}

func (m *cryptographySchemeMock) DataSigner(privateKey crypto.PrivateKey, hasher cryptography.Hasher) cryptography.Signer {
	panic("not implemented")
}

func (m *cryptographySchemeMock) DigestSigner(privateKey crypto.PrivateKey) cryptography.Signer {
	panic("not implemented")
}

func (m *cryptographySchemeMock) DataVerifier(publicKey crypto.PublicKey, hasher cryptography.Hasher) cryptography.Verifier {
	panic("not implemented")
}

func (m *cryptographySchemeMock) DigestVerifier(publicKey crypto.PublicKey) cryptography.Verifier {
	panic("not implemented")
}

func (m *cryptographySchemeMock) PublicKeySize() int {
	panic("not implemented")
}

func (m *cryptographySchemeMock) SignatureSize() int {
	panic("not implemented")
}

func (m *cryptographySchemeMock) ReferenceHashSize() int {
	panic("not implemented")
}

func (m *cryptographySchemeMock) IntegrityHashSize() int {
	panic("not implemented")
}

func NewPlatformCryptographyScheme() cryptography.PlatformCryptographyScheme {
	return &cryptographySchemeMock{}
}

type SyncT struct {
	testing.TB

	mu sync.Mutex
}

var _ testing.TB = (*SyncT)(nil)

func (t *SyncT) Error(args ...interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.TB.Error(args...)
}
func (t *SyncT) Errorf(format string, args ...interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.TB.Errorf(format, args...)
}
func (t *SyncT) Fail() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.TB.Fail()
}
func (t *SyncT) FailNow() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.TB.FailNow()
}
func (t *SyncT) Failed() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.TB.Failed()
}
func (t *SyncT) Fatal(args ...interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.TB.Fatal(args...)
}
func (t *SyncT) Fatalf(format string, args ...interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.TB.Fatalf(format, args...)
}
func (t *SyncT) Log(args ...interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.TB.Log(args...)
}
func (t *SyncT) Logf(format string, args ...interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.TB.Logf(format, args...)
}
func (t *SyncT) Name() string {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.TB.Name()
}
func (t *SyncT) Skip(args ...interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.TB.Skip(args...)
}
func (t *SyncT) SkipNow() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.TB.SkipNow()
}
func (t *SyncT) Skipf(format string, args ...interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.TB.Skipf(format, args...)
}
func (t *SyncT) Skipped() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.TB.Skipped()
}
func (t *SyncT) Helper() {}
