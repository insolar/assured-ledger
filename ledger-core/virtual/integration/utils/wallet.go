// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package utils

import (
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type Wallet struct {
	Balance uint32
}

func (w Wallet) Serialize() []byte {
	walletBytes, err := insolar.Serialize(w)
	if err != nil {
		panic(throw.W(err, "failed to serialize wallet"))
	}
	return walletBytes
}

func DeserializeWallet(walletBytes []byte) Wallet {
	wallet := Wallet{}
	insolar.MustDeserialize(walletBytes, &wallet)
	return wallet
}

func CreateWallet(balance uint32) []byte {
	return Wallet{Balance: balance}.Serialize()
}

func SerializeCreateWalletResultOK(objectRef reference.Global) []byte {
	resultBytes, err := insolar.Serialize([]interface{}{objectRef, error(nil)})
	if err != nil {
		panic(throw.W(err, "failed to serialize OK result"))
	}
	return resultBytes
}
