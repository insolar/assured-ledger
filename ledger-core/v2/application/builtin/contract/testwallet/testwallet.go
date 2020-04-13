// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package testwallet

import (
	"errors"
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/builtin/foundation"
)

// Wallet - basic wallet contract.
type Wallet struct {
	foundation.BaseContract
	balance uint32
}

const initialBalance = 1000000000

// New creates new wallet.
func New() (*Wallet, error) {
	return &Wallet{balance: initialBalance}, nil
}

// ins:immutable
func (w *Wallet) Balance() (uint32, error) {
	return w.balance, nil
}

func (w *Wallet) Accept(amount uint32) error {
	w.balance += amount
	return nil
}

func (w *Wallet) Transfer(toWallet insolar.Reference, amount uint32) error {
	if amount > w.balance {
		return errors.New("wallet balance doesn't have enough amount")
	}

	proxyWallet := testwallet.GetObject(toWallet)
	if proxyWallet == nil {
		return errors.New("toWallet is not object reference")
	}

	err := proxyWallet.Accept(amount)
	if err != nil {
		return fmt.Errorf("toWallet failed to accept trasfer with error: %s", err.Error())
	}

	w.balance -= amount

	return nil
}
