package testutils

import (
	"crypto"
	"hash"
	"sync"
	"testing"

	"golang.org/x/crypto/sha3"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
)

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
