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
