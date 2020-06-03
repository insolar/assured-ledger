// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package merkle

import (
	"encoding/binary"
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
)

const reserved = 0xDEADBEEF

func uInt32ToBytes(n uint32) []byte {
	buff := make([]byte, 4)
	binary.BigEndian.PutUint32(buff, n)
	return buff
}

type merkleHelper struct {
	scheme     cryptography.PlatformCryptographyScheme
	leafHasher cryptography.Hasher
}

func newMerkleHelper(scheme cryptography.PlatformCryptographyScheme) *merkleHelper {
	return &merkleHelper{
		scheme:     scheme,
		leafHasher: scheme.IntegrityHasher(),
	}
}

func (mh *merkleHelper) doubleSliceHash(slice1, slice2 []byte) []byte {
	hasher := mh.scheme.IntegrityHasher()
	var err error

	_, err = hasher.Write(slice1)
	if err != nil {
		panic(fmt.Sprintf("[ doubleSliceHash ] Hash write error: %s", err.Error()))
	}
	_, err = hasher.Write(slice2)
	if err != nil {
		panic(fmt.Sprintf("[ doubleSliceHash ] Hash write error: %s", err.Error()))
	}

	return hasher.Sum(nil)
}

func (mh *merkleHelper) pulseHash(pulse *pulsestor.Pulse) []byte {
	pulseNumberHash := mh.leafHasher.Hash(pulse.PulseNumber.Bytes())
	entropyHash := mh.leafHasher.Hash(pulse.Entropy[:])

	return mh.doubleSliceHash(pulseNumberHash, entropyHash)
}

func (mh *merkleHelper) nodeInfoHash(pulseHash, stateHash []byte) []byte {
	return mh.doubleSliceHash(pulseHash, stateHash)
}

func (mh *merkleHelper) nodeHash(nodeSignature, nodeInfoHash []byte) []byte {
	nodeSignatureHash := mh.leafHasher.Hash(nodeSignature)
	return mh.doubleSliceHash(nodeSignatureHash, nodeInfoHash)
}

func (mh *merkleHelper) bucketEntryHash(entryIndex uint32, nodeHash []byte) []byte {
	entryIndexHash := mh.leafHasher.Hash(uInt32ToBytes(entryIndex))
	return mh.doubleSliceHash(entryIndexHash, nodeHash)
}

func (mh *merkleHelper) bucketInfoHash(role node.StaticRole, nodeCount uint32) []byte {
	roleHash := mh.leafHasher.Hash(uInt32ToBytes(uint32(role)))
	nodeCountHash := mh.leafHasher.Hash(uInt32ToBytes(nodeCount))
	return mh.doubleSliceHash(roleHash, nodeCountHash)
}

func (mh *merkleHelper) bucketHash(bucketInfoHash, bucketEntryHash []byte) []byte {
	return mh.doubleSliceHash(bucketInfoHash, bucketEntryHash)
}

func (mh *merkleHelper) globuleInfoHash(prevCloudHash []byte, globuleID, nodeCount uint32) []byte {
	reservedHash := mh.leafHasher.Hash(uInt32ToBytes(reserved))
	globuleIDHash := mh.leafHasher.Hash(uInt32ToBytes(globuleID))
	nodeCountHash := mh.leafHasher.Hash(uInt32ToBytes(nodeCount))

	return mh.doubleSliceHash(
		mh.doubleSliceHash(reservedHash, prevCloudHash),
		mh.doubleSliceHash(globuleIDHash, nodeCountHash),
	)
}

func (mh *merkleHelper) globuleHash(globuleInfoHash, globuleNodeRoot []byte) []byte {
	return mh.doubleSliceHash(globuleInfoHash, globuleNodeRoot)
}
