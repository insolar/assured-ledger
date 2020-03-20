// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package apinetwork

// ShortNodeID is the shortened ID of node that is unique inside the globe
type ShortNodeID uint32 // ZERO is RESERVED

const AbsentShortNodeID ShortNodeID = 0

func (v ShortNodeID) IsAbsent() bool { return v == AbsentShortNodeID }

// deprecated
func (v ShortNodeID) Equal(other ShortNodeID) bool { return v == other }
