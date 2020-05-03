// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reference

import "github.com/insolar/assured-ledger/ledger-core/v2/pulse"

type LocalHeader uint32

func NewLocalHeader(pn pulse.Number, scope SubScope) LocalHeader {
	return LocalHeader(pn.WithFlags(int(scope)))
}

func (v LocalHeader) Pulse() pulse.Number {
	return pulse.OfUint32(uint32(v))
}

func (v LocalHeader) SubScope() SubScope {
	return SubScope(pulse.FlagsOf(uint32(v)))
}

func (v LocalHeader) WithPulse(pn pulse.Number) LocalHeader {
	return LocalHeader(pn.WithFlags(int(v.SubScope())))
}

func (v LocalHeader) WithSubScope(scope SubScope) LocalHeader {
	return LocalHeader(v.Pulse().WithFlags(int(scope)))
}
