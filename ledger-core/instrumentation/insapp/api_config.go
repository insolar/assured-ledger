// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insapp

import (
	"crypto"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	inscrypto "github.com/insolar/assured-ledger/ledger-core/crypto"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
)

type CertManagerFactoryFunc = func(crypto.PublicKey, cryptography.KeyProcessor, string) (*mandates.CertificateManager, error)
type KeyStoreFactoryFunc = func(string) (cryptography.KeyStore, error)

type ConfigurationProvider interface {
	GetDefaultConfig() configuration.Configuration
	GetNamedConfig(string) configuration.Configuration
	GetKeyStoreFactory() KeyStoreFactoryFunc
	GetCertManagerFactory() CertManagerFactoryFunc
}

type PreComponents struct {
	CryptographyService        cryptography.Service
	PlatformCryptographyScheme cryptography.PlatformCryptographyScheme
	KeyStore                   cryptography.KeyStore
	KeyProcessor               cryptography.KeyProcessor
	CryptoScheme               inscrypto.PlatformScheme
	CertificateManager         nodeinfo.CertificateManager
}

