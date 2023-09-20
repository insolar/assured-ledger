package vnlmn

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/rms"
)

type ResultAwaitKey struct {
	AnticipatedRef rms.Reference
	RequiredFlag   rms.RegistrationFlags
}

func NewResultAwaitKey(ref rms.Reference, requiredFlag rms.RegistrationFlags) ResultAwaitKey {
	return ResultAwaitKey{
		AnticipatedRef: ref,
		RequiredFlag:   requiredFlag,
	}
}

func (k *ResultAwaitKey) String() string {
	return fmt.Sprintf("[ %s - %s ]", k.AnticipatedRef.GetValue().String(), k.RequiredFlag.String())
}
