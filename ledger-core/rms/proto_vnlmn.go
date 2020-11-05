// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

const (
	TypeLVerifyRequestPolymorphID            = TypeLRegisterRequestPolymorphID + 1
)

type (
	LVerifyRequest            = LRegisterRequest
)

func init() {
	RegisterMessageType(TypeLVerifyRequestPolymorphID, "", (*LVerifyRequest)(nil))
}

