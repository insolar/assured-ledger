// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

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
