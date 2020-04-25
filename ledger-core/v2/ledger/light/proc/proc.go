// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package proc

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/configuration"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger/light/executor"
)

type Dependencies struct {
	WaitHot     func(*WaitHot)
	CalculateID func(*CalculateID)
	Config      func() configuration.Ledger
}

func NewDependencies(
	// Common components.
	pcs insolar.PlatformCryptographyScheme,
	hotWaiter executor.JetWaiter,

	config configuration.Ledger,
) *Dependencies {
	dep := &Dependencies{
		WaitHot: func(p *WaitHot) {
			p.Dep(
				hotWaiter,
			)
		},
		CalculateID: func(p *CalculateID) {
			p.Dep(pcs)
		},
		Config: func() configuration.Ledger {
			return config
		},
	}
	return dep
}

// NewDependenciesMock returns all dependencies for handlers.
// It's all empty.
// Use it ONLY for tests.
func NewDependenciesMock() *Dependencies {
	return &Dependencies{
		WaitHot:     func(*WaitHot) {},
		CalculateID: func(*CalculateID) {},
		Config:      configuration.NewLedger,
	}
}
