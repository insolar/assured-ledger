// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package readbundle

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

// BasicReader provides basic read access.
// WARNING! Caller MUST NOT change the byte slice.
type BasicReader interface {
	// GetDirectoryEntry returns key and entry's locator for the given index. Invalid index will return (nil, 0)
	GetDirectoryEntryLocator(ledger.DirectoryIndex) (ledger.StorageLocator, error)
	// GetEntryStorage returns start of byte slice for the given locator. Invalid locator will return nil.
	GetEntryStorage(ledger.StorageLocator) (Slice, error)
	// GetPayloadStorage returns start of byte slice for the given locator and size. Invalid locator will return nil.
	// WARNING! Implementation MAY NOT check if size is longer than the actual content for the given locator.
	GetPayloadStorage(ledger.StorageLocator, int) (Slice, error)
}

type Reader interface {
	BasicReader
	FindProvider

	FindDirectoryEntryLocator(ledger.SectionID, reference.Holder) (ledger.StorageLocator, error)
	FindDirectoryEntry(ledger.SectionID, reference.Holder) (ledger.Ordinal, error)
}

type FindProvider interface {
	FinderOfNext(ledger.SectionID) DirectoryIndexFinder
	FinderOfFirst(ledger.SectionID) DirectoryIndexFinder
	FinderOfLast(ledger.SectionID) DirectoryIndexFinder
	// FilamentFinder
}

type DirectoryIndexFinder interface {
	LookupByIndex(ledger.DirectoryIndex) (ledger.DirectoryIndex, error)
}

// type DirectoryIndexFinder interface {
// 	LookupByIndex(ledger.DirectoryIndex) (ledger.DirectoryIndex, error)
// }

type Slice interface {
	longbits.FixedReader
}

func WrapBytes(b []byte) Slice {
	return longbits.WrapBytes(b)
}

type Unmarshaler interface {
	Unmarshal([]byte) error
}

func UnmarshalTo(from io.WriterTo, to Unmarshaler) error {
	return UnmarshalToFunc(from, to.Unmarshal)
}

func UnmarshalToFunc(from io.WriterTo, toFn func ([]byte) error) error {
	_, err := from.WriteTo(writeToFunc(toFn))
	return err
}

type writeToFunc func ([]byte) error

func (v writeToFunc) Write(b []byte) (int, error) {
	if err := v(b); err != nil {
		return 0, err
	}
	return len(b), nil
}
