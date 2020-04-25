// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package proc

import (
	"context"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger/light/executor"
)

type WaitHot struct {
	jetID   insolar.JetID
	pulse   insolar.PulseNumber
	message payload.Meta

	dep struct {
		waiter executor.JetWaiter
	}
}

func NewWaitHot(j insolar.JetID, pn insolar.PulseNumber, msg payload.Meta) *WaitHot {
	return &WaitHot{
		jetID:   j,
		pulse:   pn,
		message: msg,
	}
}

func (p *WaitHot) Dep(
	w executor.JetWaiter,
) {
	p.dep.waiter = w
}

func (p *WaitHot) Proceed(ctx context.Context) error {
	return p.dep.waiter.Wait(ctx, p.jetID, p.pulse)
}

type CalculateID struct {
	payload []byte
	pulse   insolar.PulseNumber

	Result struct {
		ID insolar.ID
	}

	dep struct {
		pcs insolar.PlatformCryptographyScheme
	}
}

func NewCalculateID(payload []byte, pulse insolar.PulseNumber) *CalculateID {
	return &CalculateID{
		payload: payload,
		pulse:   pulse,
	}
}

func (p *CalculateID) Dep(pcs insolar.PlatformCryptographyScheme) {
	p.dep.pcs = pcs
}

func (p *CalculateID) Proceed(ctx context.Context) error {
	h := p.dep.pcs.ReferenceHasher()
	_, err := h.Write(p.payload)
	if err != nil {
		return errors.Wrap(err, "failed to calculate id")
	}

	p.Result.ID = *insolar.NewID(p.pulse, h.Sum(nil))
	return nil
}
