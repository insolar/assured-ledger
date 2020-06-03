// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniproto

type Flags uint8

const (
	DatagramOnly Flags = 1 << iota
	DatagramAllowed
	DisableRelay
	OmitSignatureOverTLS
	SourcePK
	OptionalTarget
	NoSourceID
)
