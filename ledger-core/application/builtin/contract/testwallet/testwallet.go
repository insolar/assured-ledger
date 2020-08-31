// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package testwallet

import (
	"errors"
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
)

// Wallet - basic wallet contract.
type Wallet struct {
	foundation.BaseContract

	Balance uint32
}

const initialBalance uint32 = 1000000000

// New creates new wallet.
func New() (*Wallet, error) {
	return &Wallet{Balance: initialBalance}, nil
}

// ins:immutable
func (w *Wallet) GetBalance() (uint32, error) {
	return w.Balance, nil
}

func (w *Wallet) Accept(amount uint32) error {
	w.Balance += amount
	return nil
}

func (w *Wallet) Transfer(toWallet reference.Global, amount uint32) error {
	if amount > w.Balance {
		return errors.New("wallet balance doesn't have enough amount")
	}

	proxyWallet := testwallet.GetObject(w.Foundation(), toWallet)
	if proxyWallet == nil {
		return errors.New("toWallet is not object reference")
	}

	err := proxyWallet.Accept(amount)
	if err != nil {
		return fmt.Errorf("toWallet failed to accept trasfer with error: %s", err.Error())
	}

	w.Balance -= amount

	return nil
}

func (w *Wallet) Destroy() error {
	return w.SelfDestruct()
}
