// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package executor

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
)

func TestJetCalculator_New(t *testing.T) {
	ctx := inslogger.TestContext(t)
	jc := jet.NewCoordinatorMock(t)
	js := jet.NewStorageMock(t)
	jetCalculator := NewJetCalculator(jc, js)
	require.NotNil(t, jetCalculator, "jet splitter created")

	me := gen.Reference()
	pn := gen.PulseNumber()
	jc.MeMock.Return(me)

	jc.LightExecutorForJetMock.Set(func(_ context.Context, jetID insolar.ID, p insolar.PulseNumber) (r *insolar.Reference, r1 error) {
		if p != pn {
			panic(fmt.Sprintf("pulse number %v is unexpected", p))
		}
		return &me, nil
	})

	var allJets []insolar.JetID
	js.AllMock.Set(func(_ context.Context, p insolar.PulseNumber) []insolar.JetID {
		if p == pn {
			return allJets
		}
		return nil
	})

	t.Run("empty_case", func(t *testing.T) {
		jets, err := jetCalculator.MineForPulse(ctx, pn)
		require.NoError(t, err)
		require.Nil(t, jets, "MineForPulse returns empty set of jets")
	})

	t.Run("one_element", func(t *testing.T) {
		allJets = []insolar.JetID{
			gen.JetID(),
		}
		jets, err := jetCalculator.MineForPulse(ctx, pn)
		require.NoError(t, err)
		require.NotNil(t, jets, "MineForPulse returns not empty set of jets")
		require.Equal(t, len(allJets), len(jets), "MineForPulse compare return values count")
		require.Equal(t, allJets, jets, "MineForPulse compare return values")
	})

	t.Run("multiple_elements", func(t *testing.T) {
		allJets = []insolar.JetID{
			jet.NewIDFromString("0"),
			jet.NewIDFromString("01"),
			jet.NewIDFromString("011"),
		}
		jets, err := jetCalculator.MineForPulse(ctx, pn)
		require.NoError(t, err)
		require.NotNil(t, jets, "MineForPulse returns not empty set of jets")
		require.Equal(t, len(allJets), len(jets), "MineForPulse compare return values count")
		require.Equal(t, allJets, jets, "MineForPulse compare return values")
	})

	t.Run("no nodes returns error", func(t *testing.T) {
		allJets = []insolar.JetID{
			jet.NewIDFromString("0"),
			jet.NewIDFromString("01"),
			jet.NewIDFromString("011"),
		}

		jc.LightExecutorForJetMock.Set(func(_ context.Context, jetID insolar.ID, p insolar.PulseNumber) (r *insolar.Reference, r1 error) {
			if p != pn {
				panic(fmt.Sprintf("pulse number %v is unexpected", p))
			}
			if insolar.JetID(jetID) == allJets[1] {
				return nil, node.ErrNoNodes
			}
			return &me, nil
		})
		_, err := jetCalculator.MineForPulse(ctx, pn)
		require.Error(t, err)
	})
}
