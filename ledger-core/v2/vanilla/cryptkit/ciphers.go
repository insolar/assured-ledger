// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package cryptkit

import "io"

type Decrypter interface {
	SignatureKey() SignatureKey
	DecryptBytes([]byte) []byte
	NewDecryptingReader(src io.Reader) io.Reader
}

type Encrypter interface {
	SignatureKey() SignatureKey
	EncryptBytes([]byte) []byte
	NewEncryptingWriter(dst io.Writer) io.Writer
}
