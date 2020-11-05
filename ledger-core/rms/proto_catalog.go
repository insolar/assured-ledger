// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"math"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func (m CatalogEntry) GetBodyAndPayloadSizes() (bodySize int, payloadSize int) {
	bodySize = int(m.BodyPayloadSizes&math.MaxUint32)
	payloadSize = int(m.BodyPayloadSizes>>32)
	return
}

func (m *CatalogEntry) SetBodySize(bodySize int) {
	switch {
	case bodySize < 0:
		panic(throw.IllegalValue())
	case bodySize > math.MaxUint32:
		panic(throw.IllegalValue())
	}

	m.BodyPayloadSizes = uint64(bodySize) | (m.BodyPayloadSizes &^ math.MaxUint32)
}

func (m *CatalogEntry) SetPayloadSize(payloadSize int) {
	switch {
	case payloadSize < 0:
		panic(throw.IllegalValue())
	case payloadSize > math.MaxUint32:
		panic(throw.IllegalValue())
	}

	m.BodyPayloadSizes = (uint64(payloadSize)<<32) | (m.BodyPayloadSizes & math.MaxUint32)
}

func (m *CatalogEntry) SetBodyAndPayloadSizes(bodySize int, payloadSize int) {
	switch {
	case bodySize < 0:
		panic(throw.IllegalValue())
	case bodySize > math.MaxUint32:
		panic(throw.IllegalValue())
	case payloadSize < 0:
		panic(throw.IllegalValue())
	case payloadSize > math.MaxUint32:
		panic(throw.IllegalValue())
	}

	m.BodyPayloadSizes = uint64(bodySize) | (uint64(payloadSize)<<32)
}
