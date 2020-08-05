// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

var _ Validatable = &VCallResult{}

func (m *VCallResult) Validate(_ PulseNumber) error {
	return nil
}
