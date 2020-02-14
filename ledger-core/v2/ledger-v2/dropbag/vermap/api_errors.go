// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package vermap

import "errors"

var (
	ErrInvalidRequest = errors.New("invalid request")
	ErrKeyNotFound    = errors.New("key not found")
	ErrEmptyKey       = errors.New("key cannot be empty")
	ErrExistingKey    = errors.New("existing key")
	ErrInvalidKey     = errors.New("key is restricted")
	ErrConflict       = errors.New("transaction conflict")
	ErrReadOnlyTxn    = errors.New("read-only transaction")
	ErrNoDelete       = errors.New("delete is not allowed")
	ErrDiscardedTxn   = errors.New("transaction has been discarded")
	ErrTxnTooBig      = errors.New("tx is too big")
)
