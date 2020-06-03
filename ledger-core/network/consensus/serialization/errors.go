// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package serialization

import (
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func ErrPayloadLengthMismatch(expected, actual int64) error {
	return errors.Errorf("payload length mismatch - expected: %d, actual: %d", expected, actual)
}

func ErrMalformedPulseNumber(err error) error {
	return errors.W(err, "malformed pulse number")
}

func ErrMalformedHeader(err error) error {
	return errors.W(err, "malformed header")
}

func ErrMalformedPacketBody(err error) error {
	return errors.W(err, "malformed packet body")
}

func ErrMalformedPacketSignature(err error) error {
	return errors.W(err, "invalid packet signature")
}
