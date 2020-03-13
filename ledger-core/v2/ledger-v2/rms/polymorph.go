/*
 * Copyright 2020 Insolar Network Ltd.
 * All rights reserved.
 * This material is licensed under the Insolar License version 1.0,
 * available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package rms

import "github.com/gogo/protobuf/proto"

var _ proto.Marshaler = Polymorph(0)

type Polymorph uint

func (p Polymorph) Marshal() ([]byte, error) {
	panic("implement me")
}

func (p Polymorph) MarshalTo(data []byte) (n int, err error) {
	panic("implement me")
}

func (p Polymorph) Unmarshal(data []byte) error {
	panic("implement me")
}

func (p Polymorph) Size() int {
	panic("implement me")
}

func (p Polymorph) MarshalJSON() ([]byte, error) {
	panic("implement me")
}

func (p Polymorph) UnmarshalJSON(data []byte) error {
	panic("implement me")
}
