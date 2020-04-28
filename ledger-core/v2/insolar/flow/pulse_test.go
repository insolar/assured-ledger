// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package flow

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/flow/internal/pulse"
)

func TestPulse(t *testing.T) {
	t.Parallel()
	ctx := pulse.ContextWith(context.Background(), 42)
	result := Pulse(ctx)
	require.Equal(t, insolar.PulseNumber(42), result)
}
