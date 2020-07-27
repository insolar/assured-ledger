// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulsartestutils

import (
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

// MockEntropy for pulsar's tests
var MockEntropy = [rms.EntropySize]byte{1, 2, 3, 4, 5, 6, 7, 8}

// MockEntropyGenerator implements EntropyGenerator and is being used for tests
type MockEntropyGenerator struct {
}

// GenerateEntropy returns mocked entropy
func (MockEntropyGenerator) GenerateEntropy() rms.Entropy {
	return MockEntropy
}
