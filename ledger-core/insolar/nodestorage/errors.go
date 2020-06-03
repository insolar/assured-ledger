// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nodestorage

import (
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var (
	// ErrOverride is returned when trying to set nodes for non-empty pulse.
	ErrOverride = errors.New("node override is forbidden")
	// ErrNoNodes is returned when nodes for specified criteria could not be found.
	ErrNoNodes = errors.New("matching nodes not found")
)
