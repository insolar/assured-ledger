package descriptor

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
)

// PrototypeDescriptor represents meta info required to fetch all prototype data.
type PrototypeDescriptor interface {
	// HeadRef returns head reference to represented object record.
	HeadRef() *insolar.Reference

	// StateID returns reference to object state record.
	StateID() *insolar.ID

	// Code returns code reference.
	Code() *insolar.Reference
}

func NewPrototypeDescriptor(
	head insolar.Reference, state insolar.ID, code insolar.Reference,
) PrototypeDescriptor {
	return &prototypeDescriptor{
		head:  head,
		state: state,
		code:  code,
	}
}

type prototypeDescriptor struct {
	head  insolar.Reference
	state insolar.ID
	code  insolar.Reference
}

// Code returns code reference.
func (d *prototypeDescriptor) Code() *insolar.Reference {
	return &d.code
}

// HeadRef returns reference to represented object record.
func (d *prototypeDescriptor) HeadRef() *insolar.Reference {
	return &d.head
}

// StateID returns reference to object state record.
func (d *prototypeDescriptor) StateID() *insolar.ID {
	return &d.state
}
