// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package catalog

import (
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsbox"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/protokit"
)

type Excerpt = rms.RecordExcerptForCatalogEntry
const MaxExcerptFieldID = 39
const excerptTagSize = 2 // protokit.WireVarint.Tag(MaxExcerptFieldID).TagSize()

func stopAfterExcerpt(b []byte) (int, error) {
	u, n := protokit.DecodeVarintFromBytes(b)
	if n != excerptTagSize {
		return 0, nil // it is something else
	}
	wt, err := protokit.SafeWireTag(u)
	if err != nil {
		return 0, err
	}
	if wt.FieldID() > MaxExcerptFieldID {
		// NB! Fields MUST be sorted
		return -1, nil // don't read other fields
	}
	return 0, nil
}

func ReadExcerptFromLazy(lazy rmsbox.LazyValueReader) (Excerpt, error) {
	var r Excerpt
	ok, err := lazy.UnmarshalAsAny(&r, stopAfterExcerpt)
	if err != nil || !ok {
		return Excerpt{}, err
	}
	return r, nil
}

func ReadExcerptFromBytes(b []byte) (Excerpt, error) {
	var r Excerpt
	if err := rmsreg.UnmarshalAs(b, &r, stopAfterExcerpt); err != nil {
		return Excerpt{}, err
	}
	return r, nil
}
