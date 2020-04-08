// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build slowtest

package grand

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils"
)

func loadMemberKeys(keysPath string) (*User, error) {
	text, err := ioutil.ReadFile(keysPath)
	if err != nil {
		return nil, errors.Wrapf(err, "[ loadMemberKeys ] could't load member keys")
	}

	var data map[string]string
	err = json.Unmarshal(text, &data)
	if err != nil {
		return nil, errors.Wrapf(err, "[ loadMemberKeys ] could't unmarshal member keys")
	}
	if data["private_key"] == "" || data["public_key"] == "" {
		return nil, errors.New("[ loadMemberKeys ] could't find any keys")
	}

	return &User{
		PrivateKey: data["private_key"],
		PublicKey:  data["public_key"],
	}, nil
}

type User struct {
	Reference        insolar.Reference
	PrivateKey       string
	PublicKey        string
	MigrationAddress string
}

func NewUserWithKeys() (*User, error) {
	ks := platformpolicy.NewKeyProcessor()

	privateKey, err := ks.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}
	privateKeyString, err := ks.ExportPrivateKeyPEM(privateKey)
	if err != nil {
		return nil, err
	}

	publicKey := ks.ExtractPublicKey(privateKey)
	publicKeyString, err := ks.ExportPublicKeyPEM(publicKey)
	if err != nil {
		return nil, err
	}

	return &User{
		PrivateKey: string(privateKeyString),
		PublicKey:  string(publicKeyString),
	}, nil
}

func NewTSAssert(tb testing.TB) *assert.Assertions {
	return assert.New(&testutils.SyncT{TB: tb})
}

func NewTSRequire(tb testing.TB) *require.Assertions {
	return require.New(&testutils.SyncT{TB: tb})
}
