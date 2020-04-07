// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package configuration

type TestWalletAPI struct {
	Address string
}

func NewTestWalletAPI() TestWalletAPI {
	return TestWalletAPI{Address: "localhost:5050"}
}
