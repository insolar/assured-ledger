// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

type StateRequestContentFlags uint32

// nolint:unused
const (
	RequestLatestValidatedState StateRequestContentFlags = 1 << iota
	RequestLatestDirtyState
	RequestLatestValidatedCode
	RequestLatestDirtyCode
	RequestMutableQueue
	RequestImmutableQueue
)

// Equal required by protobuf custom type
func (f StateRequestContentFlags) Equal(other StateRequestContentFlags) bool {
	return f == other
}

func (f StateRequestContentFlags) Contains(other StateRequestContentFlags) bool {
	return f^other == 0
}
