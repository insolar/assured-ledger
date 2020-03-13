package rms

import "github.com/insolar/assured-ledger/ledger-core/v2/reference"

type RecordReference interface {
	GoGoMarshaller
	Reference() reference.Holder
	CalcReference() reference.Holder
}
