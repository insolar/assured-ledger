// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package catalog

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

var _ rms.MarshalerTo = &rms.CatalogEntry{}
var _ bundle.MarshalerTo = &rms.CatalogEntry{}
type Entry = rms.CatalogEntry
