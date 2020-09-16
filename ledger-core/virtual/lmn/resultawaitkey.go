// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lmn

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/rms"
)

type ResultAwaitKey struct {
	AnticipatedRef rms.Reference
	RequiredFlag   rms.RegistrationFlags
}

func NewResultAwaitKey(ref rms.Reference, RequiredFlag rms.RegistrationFlags) ResultAwaitKey {
	return ResultAwaitKey{
		AnticipatedRef: ref,
		RequiredFlag:   RequiredFlag,
	}
}

func (k *ResultAwaitKey) String() string {
	return fmt.Sprintf("[ %s - %s ]", k.AnticipatedRef.GetValue().String(), k.RequiredFlag.String())
}
