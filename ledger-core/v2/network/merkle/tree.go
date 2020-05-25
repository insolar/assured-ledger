// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package merkle

import (
	"hash"

	"github.com/onrik/gomerkle"

	errors "github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type tree interface {
	Root() []byte
}

func treeFromHashList(list [][]byte, hasher hash.Hash) (tree, error) {
	mt := gomerkle.NewTree(hasher)
	mt.AddHash(list...)

	if err := mt.Generate(); err != nil {
		return nil, errors.W(err, "[ treeFromHashList ] Failed to generate merkle tree")
	}

	return &mt, nil
}
