// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package cryptkit

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit.CertificateHolder -o . -s _mock.go -g

type CertificateHolder interface {
	GetPublicKey() SigningKeyHolder
	IsValidForHostAddress(HostAddress string) bool
}
