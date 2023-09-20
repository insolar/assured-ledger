package cryptkit

import "io"

type Decrypter interface {
	SignatureKey() SigningKey
	DecryptBytes([]byte) []byte
	NewDecryptingReader(src io.Reader, encryptedSize uint) (r io.Reader, plainSize uint)
}

type Encrypter interface {
	SignatureKey() SigningKey
	EncryptBytes([]byte) []byte
	NewEncryptingWriter(dst io.Writer, plainSize uint) io.Writer

	GetOverheadSize(dataSize uint) uint
}
