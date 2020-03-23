/*
 * Copyright 2020 Insolar Network Ltd.
 * All rights reserved.
 * This material is licensed under the Insolar License version 1.0,
 * available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
)

type PlatformCryptographyProvider interface {
	PlatformCryptographyProvider()

	GetPlatformCryptographyScheme() PlatformCryptographyScheme
	GetExtensionCryptographyScheme(ExtensionId) CryptographyScheme
}

type CryptographyScheme interface {
	CryptographyScheme()
}

type PlatformCryptographyScheme interface {
	CryptographyScheme
	GetRecordBodySigner() cryptkit.DataSigner
}

type PayloadProvider interface {
	GetPayloadContainer(CryptographyScheme) GoGoMarshaller
}

type ExtensionId uint16
type ExtensionProvider struct {
	Provider  PayloadProvider
	Extension ExtensionId
}
